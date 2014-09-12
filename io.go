package main

import (
	// "fmt"
	"io/ioutil"
	"log"
	"sync/atomic"
)

type io_resp struct {
	dir  string
	data []byte
}

func new_reader(c_query_reader chan string, f_query_reader chan int32) {
	var send_in_tmp, send_in_count, send_out_count, send_gofiles int32 = 0, 0, 0, -1

	go func() {
		var file string
		var ok bool
	L:
		for {
			select {
			case file, ok = <-c_query_reader:
				if !ok {
					break L
				}
				atomic.AddInt32(&send_in_tmp, 1)

				if !is_exist(file) {
					log.Println("new_reader os.Stat() failed")
					break
				}

				data, err := ioutil.ReadFile(file)
				if err != nil {
					log.Println("can not read file")
					break
				}

				c_reader_scan <- &io_resp{file, data}
				atomic.AddInt32(&send_out_count, 1)

				atomic.StoreInt32(&send_in_count, send_in_tmp)

			case all := <-f_query_reader:
				atomic.StoreInt32(&send_gofiles, all)
			}

			if send_in_count == send_gofiles {
				//test
				// fmt.Println("io finished:------------------", send_out_count, "in", send_in_count)

				f_reader_scan <- send_out_count

				atomic.StoreInt32(&send_in_count, 0)
				atomic.StoreInt32(&send_gofiles, -1)
				//rewrite the storage if anything update
				g_wg.Add(1)
				c_storage_rewrite <- true
			}
		}
	}()
}
