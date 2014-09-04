package main

import (
	// "bufio"
	"flag"
	"fmt"
	"log"
	"os"
	// "strings"
	"time"

	"github.com/mitchellh/colorstring"
)

func main() {
	t := time.Now()
	flag.Usage = usage
	flag.Parse()

	once(t)

	//wait for the rewrite of the configfile.
	wg.Wait()
}

func once(t time.Time) int {
	query.set(os.Args[1:])
	count := lookup()

	if check_return(count) {
		log.Println("======== find sum(", count, ") finished in", time.Since(t), "========")
	}
	return 0
}

// func more() int {
// 	var r = bufio.NewReader(os.Stdin)
// 	var line string
// 	var err error

// 	for {
// 		if query.out == nil {
// 			fmt.Print(">>>")
// 		} else {
// 			fmt.Fprint(query.out, ">>>")
// 		}

// 		line, err = r.ReadString('\n')
// 		if err != nil {
// 			log.Println(err)
// 			continue
// 		}

// 		if len(line) == 0 {
// 			continue
// 		}

// 		if strings.TrimSpace(line) == "q" || strings.TrimSpace(line) == "quit" {
// 			return 0
// 		}

// 		args := strings.Fields(line)
// 		query.set(args)

// 		log.Println(query.pkg, query.name)

// 		cout := lookup()

// 		if check_return(cout) {
// 			fmt.Fprintln(query.out, "find sum(", cout, ")")
// 		}
// 	}
// 	return 0
// }

func lookup() int {
	c_main_query <- query.pkg
	count := <-c_scan_main

	return int(count)
}

func check_return(count int) bool {
	//if find nothing, new_storage will scan the goroot && gopaths again with newest data
	if count == 0 && findpkgs == 0 {
		g_st = new_storage()
		count = lookup()
		if count != 0 {
			c_storage_rewrite <- true
			return true
		}
		fmt.Fprintln(query.out,
			colorstring.Color(fmt.Sprintf("[yellow]can't find [red]%q [yellow]in the GOROOT && GOPATHS", query.pkg)))

		rcmd := g_st.recomand(query.pkg)
		if rcmd != nil {
			fmt.Fprintln(query.out, colorstring.Color("[cyan]it maybe to be:"))
			for k, v := range rcmd {
				fmt.Fprintf(query.out, colorstring.Color(fmt.Sprintf("[green]%q\t[yellow]----%s\n", diff(query.pkg, v.name), v.path)))
				if k > 10 {
					break
				}
			}
		}

		return false
	}
	return true
}
