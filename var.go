package main

import (
	"log"
	"runtime"
	"sync"
)

var (
	g_st  *storage
	query Query
)

var (
	err      error
	g_numCPU = runtime.NumCPU()
	gopaths  = get_gopaths()

	findpkgs int32
	g_wg     = &sync.WaitGroup{}
)

var (
	c_main_query = make(chan string, g_numCPU*2+2)

	c_query_reader chan string = make(chan string, g_numCPU*2+2)
	f_query_reader chan int32  = make(chan int32)

	c_reader_scan chan *io_resp = make(chan *io_resp, g_numCPU*2+2)
	f_reader_scan chan int32    = make(chan int32)

	c_scan_main chan int32 = make(chan int32)

	c_storage_rewrite chan bool = make(chan bool)
)

func init() {
	runtime.GOMAXPROCS(g_numCPU)

	g_st, err = read_from_file(config_file())
	if err != nil {
		log.Println("can't read for config file,we will rebuild it", err)
		g_st = new_storage()
		g_st.rewrite = true
	}
	g_st.write()

	g_st.query(c_main_query)
	new_reader(c_query_reader, f_query_reader)
	scan(c_reader_scan, f_reader_scan)
}
