package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func is_exist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}

func get_gopaths() []string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		panic("you should set 'GOPATH' in the environment!")
	}

	var sep string

	switch runtime.GOOS {
	case "windows":
		if strings.HasSuffix(gopath, ";") {
			gopath = gopath[:len(gopath)-1]
		}
		sep = ";"
	default:
		sep = ":"
	}
	paths := strings.Split(gopath, sep)
	//add '/src' so will can get the pkg directly
	for k, v := range paths {
		paths[k] = filepath.Join(v, "src")
	}
	return paths
}

func goversion() string {
	return fmt.Sprintf("%s %s_%s", runtime.Version(), runtime.GOARCH, runtime.GOOS)
}

//for the goroot dir -> "0" and "0"->dir
//path[i]->"i" and i begin with 1
func get_code() map[string]string {
	var m = make(map[string]string)
	goroot := filepath.Join(runtime.GOROOT(), "src/pkg")
	m["0"] = goroot
	m[goroot] = "0"

	var tmp string
	for k, v := range gopaths {
		tmp = strconv.Itoa(k + 1)

		m[tmp] = v
		m[v] = tmp
	}
	return m
}

//filter the dir not to scan
func filter(fi os.FileInfo) bool {
	name := fi.Name()
	if name == "testing" {
		return false
	}

	if strings.Contains(name, "test") || strings.HasPrefix(name, ".") ||
		name == "static" || name == "css" || name == "js" || name == "img" ||
		name == "images" || name == "fonts" {
		return true
	}
	return false
}

func filter_gofile(fi os.FileInfo) bool {
	if strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go") {
		return true
	}
	return false
}

//time to stamptime string
func stamptime(t time.Time) string {
	if t.IsZero() {
		log.Println("time is Zero")
		return ""
	}
	//because the update maybe take several seconds
	//make the files diff time with several seconds
	t = t.Add(5 * time.Second)
	return t.Format("2006-01-02 15:04:05 -0700")
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s [package.][struct.]name [outToFile]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,
		"\n 	package.      search content in the package"+
			"\n 	name          the name you want to know"+
			"\n 	outToFile     the file you want the result output to if nil will print on console\n")
}

//pointer for the sort
//get the most similar string with the package name
type pointer struct {
	name  string
	path  string
	point float64
}

type ps []*pointer

func (this ps) Len() int {
	return len(this)
}

func (this ps) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this ps) Less(i, j int) bool {
	return this[i].point+math.SmallestNonzeroFloat64 > this[j].point
}

//get the pointer src similar to the dst
func similar(dst, src string) float64 {
	num_src := len(src)
	num_dst := len(dst)

	if num_src == 0 || num_dst == 0 {
		return 0
	}

	index_slice := make([]int, num_src)
	var prev, slice int

	for k, v := range src {
		dst = dst[slice:]
		index := strings.IndexRune(dst, v)
		if index == -1 {
			return 0
		}
		slice = index + 1
		index_slice[k] = index + prev
		prev = prev + index + 1
		if prev > num_dst {
			return 0
		}
	}
	var point_sum float64

	for i := 1; i < num_src; i++ {
		point_sum += (1 / float64(index_slice[i]-index_slice[i-1]))
	}

	//the float64(num_src) to average the every char's weight
	//把每个字母的权重平均化
	point_sum += (float64(num_src) / float64(num_dst)) * float64(num_src)

	return point_sum / float64(num_src*2-1)
}

//give parently diff in string
//trng --> strings
func diff(input, target string) []byte {
	inputs := len(input)
	idx := make([]int, inputs)

	tmp := target
	var index, prev int

	for k, r := range input {
		index = strings.IndexRune(tmp, r)
		tmp = tmp[index+1:]
		idx[k] = prev + index
		prev += index + 1
	}

	slice := make([]string, 0, inputs)
	var start, end = idx[0], idx[0]
	for k := 1; k < inputs; k++ {
		if idx[k]-idx[k-1] > 1 {
			slice = append(slice, target[start:end+1])
			start, end = idx[k], idx[k]
			continue
		}
		end++
	}
	slice = append(slice, target[start:end+1])
	//new
	tmp = target
	index = -1
	src := ""

	if idx[0] > 0 {
		src = "[green]" + tmp[:idx[0]]
		prev = idx[0]
		tmp = tmp[idx[0]:]
	}

	for _, v := range slice {
		index = strings.Index(tmp, v)
		if index == 0 {
			src = src + "[red]" + tmp[index:index+len(v)]
			prev = index + len(v)
			tmp = tmp[prev:]
		} else {
			src = src + "[green]" + tmp[:index] + "[red]" + tmp[index:index+len(v)]
			tmp = tmp[index+len(v):]
		}
	}
	if len(tmp) > 0 {
		src = src + "[green]" + tmp
	}

	return []byte(src)
}
