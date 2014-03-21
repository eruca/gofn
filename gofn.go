//多纤程版本，可是在runtime.GOMAXPROCS()不调用的时候性能更高
//那就写一个单线程的吧
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	//"runtime"
	"strings"
	"sync"
	"time"
)

var Stdpkgs = [...]string{"hash/adler32", "image/png", "math/cmplx", "net/url", "path", "runtime/debug", "testing/iotest", "bufio", "crypto", "crypto/des", "crypto/rand", "encoding/json", "go/ast", "runtime/cgo", "text/template/parse", "builtin", "cmd/api", "crypto/rc4", "regexp", "sync/atomic", "unicode", "debug/gosym", "encoding/base64", "net/http/httptest", "os/exec", "testing/quick", "encoding/hex", "go/scanner", "image/draw", "runtime", "text/scanner", "cmd/yacc", "crypto/elliptic", "runtime/race", "cmd/fix", "cmd/vet", "container/list", "crypto/sha256", "net/http/cgi", "os/user", "crypto/subtle", "go/parser", "net/rpc", "path/filepath", "runtime/pprof", "crypto/cipher", "crypto/dsa", "crypto/sha512", "crypto/x509", "debug/dwarf", "debug/elf", "go/printer", "image/gif", "regexp/syntax", "crypto/md5", "net/textproto", "cmd/gofmt", "container/ring", "encoding/base32", "mime/multipart", "os/signal", "crypto/sha1", "fmt", "net/mail", "bytes", "encoding/csv", "encoding/xml", "go/format", "html/template", "text/template", "compress/flate", "go/doc", "io/ioutil", "syscall", "unicode/utf8", "compress/bzip2", "compress/zlib", "encoding/pem", "hash/fnv", "mime", "crypto/tls", "crypto/x509/pkix", "net", "time", "container/heap", "encoding/asn1", "encoding/binary", "hash/crc32", "crypto/rsa", "debug/macho", "log", "unicode/utf16", "database/sql", "errors", "compress/gzip", "math", "compress/lzw", "crypto/hmac", "math/rand", "net/http/cookiejar", "encoding/ascii85", "io", "net/rpc/jsonrpc", "strings", "crypto/ecdsa", "html", "cmd/go", "database/sql/driver", "text/tabwriter", "expvar", "go/build", "image/color", "net/http/fcgi", "sync", "encoding/gob", "os", "sort", "archive/zip", "testing", "cmd/cgo", "debug/pe", "go/token", "reflect", "strconv", "hash", "image/jpeg", "net/smtp", "unsafe", "archive/tar", "crypto/aes", "hash/crc64", "log/syslog", "math/big", "flag", "image", "index/suffixarray", "net/http", "net/http/httputil", "net/http/pprof"}

type Search struct {
	Finder []string
	Name   string
	Mutex  *sync.Mutex
}

type FindPath struct {
	targetPkg string
	find      []string
}

var g_search = &Search{Finder: make([]string, 0, 256), Mutex: new(sync.Mutex)}
var g_find = &FindPath{find: make([]string, 0, 32)}

func main() {
	t := time.Now()
	//runtime.GOMAXPROCS(runtime.NumCPU())

	var pkg, name string
	//需要处理有参数的情况，如果没有可能会panic
	pkgAndName := strings.Split(os.Args[1], ".")

	if len(pkgAndName) == 0 || len(pkgAndName) > 2 {
		log.Fatalln("the param 1 is not in right fomat")
	} else if len(pkgAndName) == 1 {
		name = pkgAndName[0]
	} else {
		pkg = pkgAndName[0]
		name = pkgAndName[1]
	}

	g_search.Name = name

	goroot := os.Getenv("GOROOT")
	if len(goroot) == 0 {
		log.Fatalln("the GOROOT is not set")
	}
	goroot = filepath.Join(goroot, "src/pkg/")

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatalln("the GOPATH is not set")
	}

	//这个是windows情况下，是不是linux或mac需要更改呢
	//如果设置2个以上GOPATH，取第一个，因为go get 默认第一个
	gopath = strings.Split(gopath, ";")[0]
	gopath = filepath.Join(gopath, "/src")

	wg := new(sync.WaitGroup)

	//判断是否预先输入package
	if len(pkg) != 0 {
		var bFind bool
		for _, v := range Stdpkgs {
			if v == pkg || strings.HasSuffix(v, "/"+pkg) {
				g_search.Finder = append(g_search.Finder, "\n------------Standard Package------------")
				bFind = true

				wg.Add(1)
				go findInFile(filepath.Join(goroot, v), wg)
				break
			}
		}
		
		if !bFind {
			g_find.targetPkg = pkg
			SearchInGoPath(gopath)
			if len(g_find.find) != 0 {
				g_search.Finder = append(g_search.Finder, "\n------------The 3rd Package------------")
			} else {
				fmt.Printf("the package %q is not find in std or 3rd package\n", pkg)
				log.Fatalln("finished in ", time.Since(t))
			}
			for _, v := range g_find.find {
				//log. the find path in the 3rd pkgs............................ need to be there
				//log.Println(v)

				wg.Add(1)
				go findInFile(v, wg)
			}
		}
	} else {
		wg.Add(1)
		go findInFile(goroot, wg)
	}

	wg.Wait()

	if len(g_search.Finder) == 1 {
		fmt.Printf("%s.%s is not found\n", pkg, name)
	} else {
		for _, v := range g_search.Finder {
			if v != "" {
				fmt.Println(v + "\n")
			}
		}
	}
	log.Println("finished in ", time.Since(t))
}

func findInFile(path string, wg *sync.WaitGroup) {
	defer wg.Done()

	fi, err := os.Stat(path)
	if err != nil {
		log.Printf("findInFile->os.Stat():%s", err.Error())
		return
	}
	result := make([]string, 0, 128)

	if fi.IsDir() {
		list, err := ioutil.ReadDir(path)
		if err != nil {
			log.Printf("findInFile->:ReadDir(%s):%s", path, err.Error())
			return
		}
		for _, fileInfo := range list {
			if fileInfo.Name()[0] == '.' {
				continue
			}
			wg.Add(1)
			go findInFile(filepath.Join(path, fileInfo.Name()), wg)
		}
	} else if filepath.Ext(fi.Name()) == ".go" {
		var lineno int
		var bFind bool
		var bCommet bool
		var comment string
		var line string
		var bTypefunc bool

		file, err := os.Open(path)
		if err != nil {
			log.Println("findInFile()->os.Open(dir) failed")
		}
		defer file.Close()

		stack := &Stack{i: 0, data: make([]int, 56)}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line = scanner.Text()
			lineno++

			if len(line) == 0 && !bFind {
				bCommet = true
				comment = ""
				continue
			} else if bCommet == true && len(line) > 2 && line[0:2] == "//" {
				comment += line + "\n"
				continue
			}

			if (len(line) > 9 && (line[0:5] == "func " || line[0:5] == "type ")) || bFind {
				var left int
				if !bFind {
					switch line[0:5] {
					case "func ":
						if left = strings.Index(line, " "+g_search.Name+"("); left == -1 {
							continue
						}
					case "type ":
						if left = strings.Index(line, " "+g_search.Name+" struct"); left == -1 {
							if left = strings.Index(line, " "+g_search.Name+" interface"); left == -1 {
								if left = strings.Index(line, " "+g_search.Name+" func("); left == -1 {
									continue
								} else {
									bTypefunc = true
								}
							}
						}
					}

					bFind = true
					bCommet = false
					result = append(result, path)
					left++
				}

				//如果函数、结构、接口内部有注释，直接跳过
				//因为reflect Type接口第43句，有注释使用中括号和小括号不对，所以才选择跳过，注释不严谨
				result = append(result, fmt.Sprintf("%d:%s", lineno, line))
				if s := strings.TrimSpace(line); len(s) > 2 && s[:2] == "//" {
					continue
				}

				//如果是定义函数别名，如 type SplitFunc func()()
				//考虑一般为自成一行
				if bTypefunc {
					bFind = false
					result = append(result, comment)
					continue
				}

				for _, v := range line[left:] {
					switch v {
					case '(':
						stack.push(T_parenthesis)
					case ')':
						//test
						if stack.pop() != T_parenthesis {
							log.Fatalln("the 'parentheis' not used in couple!")
						}
					case '{':
						if stack.i == 0 {
							stack.bUse = true
						}
						stack.push(T_brace)
					case '}':
						if stack.pop() != T_brace {
							log.Fatalln("the 'brace' not used in couple")
						}
						if stack.bUse && stack.i == 0 && line[len(line)-1] == '}' {
							bFind = false
							//put commet add to the end
							result = append(result, comment)
							break
						}
					}
				}
			}
		}
	}

	if len(result) != 0 {
		g_search.Mutex.Lock()
		g_search.Finder = append(g_search.Finder, strings.Join(result, "\n"))
		g_search.Mutex.Unlock()
	}
}

func SearchInGoPath(root string) {
	if g_find.targetPkg == "" {
		log.Println("the target pkg is nil")
		return
	}

	fi, err := os.Stat(root)
	if err != nil {
		log.Printf("SearchInGoPath->os.Stat():%s", err.Error())
		return
	}
	if !fi.IsDir() {
		log.Printf("the param %s is not a dir", root)
		return
	}

	list, err := ioutil.ReadDir(root)
	if err != nil {
		log.Printf("SearchInGoPath->: ReadDir(%s):%s\n", root, err.Error())
		return
	}

	for _, fileInfo := range list {
		if !fileInfo.IsDir() || fileInfo.Name()[0] == '.' {
			continue
		}

		if fileInfo.Name() == g_find.targetPkg {
			g_find.find = append(g_find.find, filepath.Join(root, fileInfo.Name()))
		} else {
			SearchInGoPath(filepath.Join(root, fileInfo.Name()))
		}
	}
}
