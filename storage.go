package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//storage-------------------------------------------------------------
type storage struct {
	Tries     []*trie
	to_update []bool
	rewrite   bool

	signal_close chan bool
}

func new_storage() *storage {
	if g_st != nil {
		g_st.close()
	}

	tries := make([]*trie, 1, 3)
	tries[0] = new_trie(filepath.Join(runtime.GOROOT(), "src/pkg"), goversion())

	for _, v := range gopaths {
		tries = append(tries, new_trie(v, "time"))
	}
	to_update := make([]bool, len(gopaths))
	signal_close := make(chan bool)

	st := &storage{Tries: tries, to_update: to_update, signal_close: signal_close}

	//st.rewrite = g_st.rewrite

	st.write()
	st.query(c_main_query)

	return st
}

func read_from_file(dir string) (*storage, error) {
	if !is_exist(dir) {
		st := new_storage()
		st.rewrite = true
		return st, nil
	}

	var this = &storage{}

	b, err := ioutil.ReadFile(dir)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, this)
	if err != nil {
		return nil, err
	}

	err = this.convert_read()
	if err != nil {
		return nil, err
	}
	this.to_update = make([]bool, len(gopaths))
	this.signal_close = make(chan bool)

	return this, nil
}

func (this *storage) same_version() bool {
	if this == nil {
		panic("the storage is nil")
	}

	if this.Tries[0].Info == goversion() {
		return true
	}
	return false
}

func (this *storage) convert_write() error {
	if this == nil {
		return errors.New("the storage is nil")
	}

	m := get_code()
	for _, v := range this.Tries {
		if _, err := strconv.Atoi(v.Root.Name); err == nil {
			continue
		}
		sub, ok := m[v.Root.Name]
		if ok {
			v.Root.Name = sub
		}
	}
	return nil
}

func (this *storage) convert_read() error {
	if this == nil {
		return errors.New("the storage is nil")
	}

	m := get_code()
	for _, v := range this.Tries {
		if _, err := strconv.Atoi(v.Root.Name); err != nil {
			continue
		}
		sub, ok := m[v.Root.Name]
		if ok {
			v.Root.Name = sub
		}
	}
	return nil
}

//add function
func (this *storage) recomand(query string) []*pointer {
	var arr = make([]*pointer, 0, 4)

	for _, v := range this.Tries {
		v.similar(query, &arr)
	}

	if len(arr) == 0 {
		return nil
	}

	sort.Sort(ps(arr))

	return arr
}

//io
func (this *storage) write() {
	if this == nil {
		panic("the storage is nil")
	}

	go func() {
		var filedir string = config_file()
		for {
			select {
			case <-c_storage_rewrite:
				//wg.Add(1)
				for k, v := range this.to_update {
					if v {
						this.rewrite = true
						this.Tries[k+1] = new_trie(gopaths[k], "time")
					}
				}

				if !this.rewrite {
					g_wg.Done()
					continue
				}

				if !is_exist(filedir) {
					os.MkdirAll(filepath.Dir(filedir), os.ModePerm)
				}

				file, err := os.Create(filedir)
				if err != nil {
					panic(err.Error())
				}

				err = this.convert_write()
				if err != nil {
					panic(err.Error())
				}

				b, err := json.Marshal(this)
				if err != nil {
					panic(err.Error())
				}

				_, err = file.Write(b)
				if err != nil {
					panic(err.Error())
				}

				log.Println("rewrite the config file")

				g_wg.Done()
			case <-this.signal_close:
				return
			}
		}
	}()
}

//new
func (this *storage) query(c_main_query chan string) {
	if this == nil {
		panic("the storage is nil")
	}

	c_pkg_gofile := make(chan string, g_numCPU*2+2)

	//保证不会死锁，下面详细说明
	a := make(chan bool)

	//b的作用是，1 goroutine 处理完了，通知 2
	b := make(chan bool)

	var send_gofiles int32
	var folds, tackled, folds_final int32 = 0, 0, -1

	for i := 0; i < g_numCPU; i++ {
		//1st goroutine
		go func() {
			var father string
			var index int = -1
			var last_update time.Time
			var err error

			for {
				select {
				case dir, ok := <-c_pkg_gofile:
					if !ok {
						break
					}

					if strings.HasPrefix(dir, father) || father == "" {
						for k, v := range gopaths {
							if strings.HasPrefix(dir, v) {
								father = v
								index = k
								last_update, err = time.Parse("2006-01-02 15:04:05 -0700", this.Tries[k+1].Info)
								if err != nil {
									log.Println("the config file's time cannot parse,we will reconstruct it again.")
									this.to_update[index] = true
									//todo
									os.Exit(2)
								}
							}
						}
					}

					fis, err := ioutil.ReadDir(dir)
					if err != nil {
						panic(err)
					}

					var gofiles int32

					for _, fi := range fis {
						if fi.IsDir() {
							continue
						}

						if index != -1 && !this.to_update[index] {
							mod := fi.ModTime()
							if mod.After(last_update) {
								this.to_update[index] = true
							}
						}

						if filter_gofile(fi) {
							c_query_reader <- filepath.Join(dir, fi.Name())
							gofiles++
						}
					}
					atomic.AddInt32(&send_gofiles, gofiles)
					atomic.AddInt32(&tackled, 1)

				//2种情况
				//
				//1.路径先发完，folds_final转变。2.文件已处理完，路径还没找完（最后不符合的文件.js .html多）
				//针对1情况基本不会有问题，最后一个路径处理好，那么两者的数字会相等
				//第2种情况，文件处理好，folds_final还没有转变，就死锁，那么a可以保持在folds_final转变后
				//再询问一次两者是否相等，如果相等那就返回
				case <-a:
					//log.Println("tackled", tackled)
					//log.Println("folds_final", folds_final)
				case <-this.signal_close:
					return
				}

				if tackled == folds_final {
					atomic.StoreInt32(&tackled, 0)
					atomic.StoreInt32(&folds, 0)
					atomic.StoreInt32(&folds_final, -1)
					b <- true
				}
			}
		}()
	}

	//2nd goroutine
	go func() {
		w := &sync.WaitGroup{}
		for {
			select {
			case pkg, ok := <-c_main_query:
				if !ok {
					break
				}

				w.Add(len(this.Tries))

				for _, v := range this.Tries {
					v.query(pkg, &folds, w, c_pkg_gofile)
				}

				w.Wait()
				atomic.StoreInt32(&folds_final, folds)

				a <- true

				//wait for 1 goroutine
				<-b

				//if did not send any gofile,goto the main goroutine immediately
				if send_gofiles == 0 {
					c_scan_main <- 0
				} else {
					//test
					//log.Println("send gofiles:", send_gofiles)

					f_query_reader <- send_gofiles
					atomic.StoreInt32(&send_gofiles, 0)
				}
			case <-this.signal_close:
				return
			}
		}
	}()
}

func (this *storage) close() {
	close(this.signal_close)
}
