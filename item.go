package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/mitchellh/colorstring"
)

//item-------------------------------------------
type item struct {
	Name     string  `json:"n"`
	Children []*item `json:"c"`
}

func new_item(name string) *item {
	return &item{Name: name}
}

func (this *item) fix_children(fis []os.FileInfo) (fix_num int, t time.Time) {
	if this == nil || len(fis) == 0 {
		return fix_num, t
	}

	var mod time.Time
	for _, fi := range fis {
		if filter(fi) {
			continue
		}
		if !fi.IsDir() {
			mod = fi.ModTime()
			if mod.After(t) {
				t = mod
			}
			continue
		}
		fix_num++
	}

	if fix_num == 0 {
		return 0, t
	}

	if this.Children == nil {
		this.Children = make([]*item, fix_num)
	} else {
		this.Children = this.Children[:fix_num]
	}

	return fix_num, t
}

func (this *item) path(p2p *safemap) string {
	v := p2p.query(unsafe.Pointer(this))
	if v == nil {
		return this.Name
	}
	it := (*item)(v)
	return filepath.Join(it.path(p2p), this.Name)
}

func (this *item) query(pkg string, folds *int32, p2p *safemap, cin chan<- string) {
	if this == nil {
		panic("the item is nil")
	}

	if this.Name == pkg || pkg == "" {
		dir := this.path(p2p)
		atomic.AddInt32(&findpkgs, 1)

		if query.name == "" && query.stru == "" {
			fmt.Fprintln(query.out, colorstring.Color(fmt.Sprintf("finds below:\n->\t[red]%s\t\t[yellow]%s", query.pkg, dir)))
			return
		}

		send_dir(dir, folds, cin)

		return
	}

	if this.Children == nil {
		return
	}

	for _, v := range this.Children {
		v.query(pkg, folds, p2p, cin)
	}
}

func send_dir(dir string, folds *int32, cout chan<- string) {
	cout <- dir
	atomic.AddInt32(folds, 1)

	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Panicln("send_dir", err)
	}

	for _, fi := range fis {
		if filter(fi) {
			continue
		}

		if fi.IsDir() {
			send_dir(filepath.Join(dir, fi.Name()), folds, cout)
		}
	}
}

func (this *item) collect(pkg string, p2p *safemap, collect *[]string) {
	if this == nil {
		panic("the item is nil")
	}

	if collect == nil {
		panic("the collect is nil")
	}

	if pkg == "" {
		return
	}

	if this.Name == pkg {
		*collect = append(*collect, this.path(p2p))
	}

	for _, v := range this.Children {
		v.collect(pkg, p2p, collect)
	}
}

func (this *item) scan(p2p *safemap) (time.Time, error) {
	if this == nil {
		return time.Time{}, errors.New("the item is nil")
	}

	path := this.path(p2p)

	fileInfo, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}

	last := fileInfo.ModTime()

	fis, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err.Error())
	}

	//just for the time (last)
	num, mod := this.fix_children(fis)
	if mod.After(last) {
		last = mod
	}
	if num == 0 {
		return last, nil
	}

	var index int
	var t time.Time

	for _, fi := range fis {
		if filter(fi) {
			continue
		}

		if !fi.IsDir() {
			t = fi.ModTime()
			if t.After(last) {
				last = t
			}
			continue
		}

		pNew := new_item(fi.Name())
		if p2p.query(unsafe.Pointer(pNew)) == nil {
			p2p.insert(unsafe.Pointer(pNew), unsafe.Pointer(this))
		}

		t, err = pNew.scan(p2p)
		if err != nil {
			panic(err.Error())
		}

		if t.After(last) {
			last = t
		}

		this.Children[index] = pNew
		index++
	}
	return last, nil
}

func (this *item) rebuild_map(p2p *safemap) error {
	if this == nil || this.Children == nil {
		return errors.New("the item or children is nil")
	}

	for _, v := range this.Children {
		if p2p.query(unsafe.Pointer(v)) == nil {
			p2p.insert(unsafe.Pointer(v), unsafe.Pointer(this))
		}
		v.rebuild_map(p2p)
	}
	return nil
}

//add can use goroutine next time
func (this *item) similar(query string, arr *[]*pointer, p2p *safemap) {
	if this == nil {
		panic("item is nil")
	}

	p := similar(this.Name, query)
	if p > 0.2 {
		*arr = append(*arr, &pointer{this.Name, this.path(p2p), p})
	}

	for _, v := range this.Children {
		v.similar(query, arr, p2p)
	}
}
