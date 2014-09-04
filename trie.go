package main

import (
	"sync"
)

//trie--------------------------------------------------------
type trie struct {
	Info string `json:"i"` //root:version path:last update time
	Root *item  `json:"r"`
	smap *safemap
}

func new_trie(path, info string) *trie {
	t := &trie{
		Root: new_item(path),
		Info: info,
		smap: new_safemap(),
	}

	err := t.scan()
	if err != nil {
		panic(err.Error())
	}
	return t
}

func (this *trie) scan() error {
	if this == nil {
		panic("the trie is nil")
	}

	t, err := this.Root.scan(this.smap)

	if this.Info == "time" {
		this.Info = stamptime(t)
	}
	return err
}

//similar
func (this *trie) similar(query string, arr *[]*pointer) {
	for _, v := range this.Root.Children {
		v.similar(query, arr, this.smap)
	}
}

//new2
func (this *trie) query(pkg string, folds *int32, wg *sync.WaitGroup, c_pkg_gofile chan<- string) {
	if this == nil {
		panic("the trie is nil")
	}

	if this.smap == nil {
		this.smap = new_safemap()
		this.Root.rebuild_map(this.smap)
	}

	go func() {
		defer wg.Done()
		this.Root.query(pkg, folds, this.smap, c_pkg_gofile)
	}()
}
