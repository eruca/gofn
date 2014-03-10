package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var pkgs = [...]string{"hash/adler32", "image/png", "math/cmplx", "net/url", "path", "runtime/debug", "testing/iotest", "bufio", "crypto", "crypto/des", "crypto/rand", "encoding/json", "go/ast", "runtime/cgo", "text/template/parse", "builtin", "cmd/api", "crypto/rc4", "regexp", "sync/atomic", "unicode", "debug/gosym", "encoding/base64", "net/http/httptest", "os/exec", "testing/quick", "encoding/hex", "go/scanner", "image/draw", "runtime", "text/scanner", "cmd/yacc", "crypto/elliptic", "runtime/race", "cmd/fix", "cmd/vet", "container/list", "crypto/sha256", "net/http/cgi", "os/user", "crypto/subtle", "go/parser", "net/rpc", "path/filepath", "runtime/pprof", "crypto/cipher", "crypto/dsa", "crypto/sha512", "crypto/x509", "debug/dwarf", "debug/elf", "go/printer", "image/gif", "regexp/syntax", "crypto/md5", "net/textproto", "cmd/gofmt", "container/ring", "encoding/base32", "mime/multipart", "os/signal", "crypto/sha1", "fmt", "net/mail", "bytes", "encoding/csv", "encoding/xml", "go/format", "html/template", "text/template", "compress/flate", "go/doc", "io/ioutil", "syscall", "unicode/utf8", "compress/bzip2", "compress/zlib", "encoding/pem", "hash/fnv", "mime", "crypto/tls", "crypto/x509/pkix", "net", "time", "container/heap", "encoding/asn1", "encoding/binary", "hash/crc32", "crypto/rsa", "debug/macho", "log", "unicode/utf16", "database/sql", "errors", "compress/gzip", "math", "compress/lzw", "crypto/hmac", "math/rand", "net/http/cookiejar", "encoding/ascii85", "io", "net/rpc/jsonrpc", "strings", "crypto/ecdsa", "html", "cmd/go", "database/sql/driver", "text/tabwriter", "expvar", "go/build", "image/color", "net/http/fcgi", "sync", "encoding/gob", "os", "sort", "archive/zip", "testing", "cmd/cgo", "debug/pe", "go/token", "reflect", "strconv", "hash", "image/jpeg", "net/smtp", "unsafe", "archive/tar", "crypto/aes", "hash/crc64", "log/syslog", "math/big", "flag", "image", "index/suffixarray", "net/http", "net/http/httputil", "net/http/pprof"}

func main() {
	if len(os.Args) != 2 && len(os.Args) != 3 {
		fmt.Println(`gofn is tool for looking for funcion of Go Standard source code.

Usage: 
	gofn [arguments]

The arguments are:` + "\n\n\t[package.]Func\tfunction name with or not the package name.	\n\t\t\twithout will search all the function.\n\ttrue/false\tfalse will search function ignoreCase.")
		os.Exit(1)
	}

	var bSame bool = true
	if len(os.Args) == 3 {
		switch strings.ToLower(os.Args[2]) {
		case "true":
			bSame = true
		case "fasle":
			bSame = false
		}
	}

	var path, name string
	pathAndName := strings.Split(os.Args[1], ".")

	if len(pathAndName) == 0 || len(pathAndName) > 2 {
		log.Fatalln("the param 1 is not in right fomat")
	} else if len(pathAndName) == 1 {
		name = pathAndName[0]
	} else {
		path = pathAndName[0]
		name = pathAndName[1]
	}

	if len(path) != 0 {
		var bFind bool
		for _, v := range pkgs {
			if v == path {
				bFind = true
				break
			} else if strings.HasSuffix(v, path) {
				bFind = true
				path = v
				break
			}
		}
		if !bFind {
			log.Fatalf("the %q is not in pkgfile", path)
		}
	}

	t := time.Now()
	goroot := os.Getenv("GOROOT")
	if len(goroot) == 0 {
		log.Fatalln("the GOROOT is not set")
	}

	gopkg := filepath.Join(goroot, "src/pkg/", path)

	trace(findFunc(gopkg, name, bSame))

	log.Println("finish in ", time.Since(t))
}

func findFunc(root, fname string, bSame bool) error {
	file, err := os.Open(root)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	s := &Stack{i: 0, data: make([]int, 56)}
	result := make([]string, 0)

	if fi.IsDir() {
		trace(filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			f, err := os.Open(path)
			if trace(err) {
				return err
			}
			defer f.Close()

			r := bufio.NewReader(f)
			var lineno int = 1
			var bFind bool
			var line string

			//把一个文件的内容收集
			var resultOfFile string

			//read line by line
			for {
				line, err = r.ReadString('\n')
				if err == io.EOF {
					err = nil
					break
				} else if err != nil {
					return err
				}

				if bFind {
					resultOfFile += fmt.Sprintf("%d: %s", lineno, line)
					for _, v := range line {
						switch v {
						case '(':
							s.push(T_parenthesis)
						case ')':
							if s.pop() != T_parenthesis {
								return errors.New("the 'parentheis' not used in couple!")
							}
						case '{':
							if s.i == 0 {
								s.bUse = true
							}
							s.push(T_brace)
						case '}':
							if s.pop() != T_brace {
								return errors.New("the 'brace' not used in couple")
							}
						}
					}
					if s.bUse && s.i == 0 {
						err = nil
						break
					}
				} else if strings.Contains(line, " "+fname+`(`) && line[0:len("func")] == "func" {
					bFind = true

					resultOfFile += fmt.Sprintf("%d: %s", lineno, line)

					left := strings.Index(line, fname)

					//delete the line end '\n'
					//log.Println("remove the left of the name is :", l)
					l := line[left+len(fname) : len(line)-1]
					//log.Println(l)

					for _, v := range l {
						switch v {
						case '(':
							s.push(T_parenthesis)
						case ')':
							if s.pop() != T_parenthesis {
								return errors.New("the 'parentheis' not used in couple!")
							}
						case '{':
							if s.i == 0 {
								s.bUse = true
							}
							s.push(T_brace)
						case '}':
							if s.pop() != T_brace {
								return errors.New("the 'brace' not used in couple")
							}
						}
					}
					if s.bUse && s.i == 0 {
						err = nil
						break
					}

				}
				lineno++
			}
			if bFind {
				resultOfFile = path + ":\n" + resultOfFile + "\n"
				result = append(result, resultOfFile)
			}
			return nil
		}))

		if len(result) != 0 {
			for _, v := range result {
				fmt.Print(v)
			}
		} else {
			log.Printf("the %q is not found!", fname)
		}
	}
	return nil
}

func trace(err error) bool {
	if err != nil {
		log.Println(err)
		return true
	}
	return false
}

const (
	T_parenthesis = 1
	T_brace       = 2
)

type Stack struct {
	i    int
	data []int
	bUse bool
}

func (s *Stack) push(n int) {
	if s.i+1 > len(s.data) {
		sint := make([]int, len(s.data))
		s.data = append(sint, s.data...)
		s.data = append(s.data, n)
	} else {
		s.data[s.i] = n
	}
	s.i++
}

func (s *Stack) pop() (n int) {
	n = s.data[s.i-1]
	s.data[s.i-1] = 0
	s.i--
	return
}
