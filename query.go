package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mitchellh/colorstring"
)

//Out----------------------------------------------
type Out struct {
	writer io.Writer
	dir    bool //false->terminal && true -> file
	m      *sync.Mutex
}

func (this *Out) Write(p []byte) (n int, err error) {
	this.m.Lock()
	defer this.m.Unlock()
	return fmt.Fprintln(this.writer, colorstring.Color(string(p)))
}

//terminal -> true otherwize false
func (this *Out) Dir() bool {
	return this.dir
}

//query----------------------------
type Query struct {
	pkg  string
	stru string
	name string

	out *Out //result output io like 'os.Stdout or file'
}

func (this *Query) set(args []string) {
	var err error
	var path string

	switch len(args) {
	case 1:
		err = this.fill(args[0])
	case 2:
		err = this.fill(args[0])
		path = args[1]
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		panic(err)
	}

	this.set_writer(path)
}

//split to pkg.[struct].function
func (this *Query) fill(pkgName string) error {
	rg := regexp.MustCompile(`[\w\._-]+`)

	if rg.MatchString(pkgName) {
		slice := strings.Split(pkgName, ".")
		switch len(slice) {
		case 1:
			this.name = slice[0]
		case 2:
			this.pkg, this.name = slice[0], slice[1]
		case 3:
			this.pkg, this.stru, this.name = slice[0], slice[1], slice[2]
		default:
			usage()
			return errors.New("the pkgname has more than 2 dots")
		}
	} else {
		usage()
		return errors.New("the pkgname contain illegal word")
	}

	return nil
}

func (this *Query) set_writer(path string) {
	out := &Out{m: new(sync.Mutex)}

	if path != "" {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Println("set_writer failed:", err)
			log.Println("we will show it in terminal!")
			path = ""
		} else {
			out.dir = false
			out.writer = file
			this.out = out

			return
		}
	}

	if path == "" {
		out.dir = true
		out.writer = colorable.NewColorableStdout()
		this.out = out
	}
}
