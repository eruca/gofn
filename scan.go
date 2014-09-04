package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/mitchellh/colorstring"
)

func scan(c_reader_scan chan *io_resp, f_reader_scan chan int32) {
	var finds_count int32
	//test
	// var test2 int32
	var mutex = sync.RWMutex{}

	var send_in_count, all int32 = 0, -1

	for i := 0; i < g_numCPU; i++ {
		go func() {
			var r *io_resp
			var ok bool
		L:
			for {
				select {
				case r, ok = <-c_reader_scan:
					if !ok {
						break L
					}

					mutex.RLock()

					vi := new_visitor(r.dir, r.data)
					if vi == nil {
						atomic.AddInt32(&send_in_count, 1)
						mutex.RUnlock()
						break
					}

					if vi.finds != nil {
						atomic.AddInt32(&finds_count, int32(len(vi.finds)))

						sort.Sort(Finds(vi.finds))

						query.out.m.Lock()
						if query.out.Dir() {
							for _, v := range vi.finds {
								fmt.Fprintln(query.out.writer, colorstring.Color(string(v.Byte_line())))
							}
						} else {
							for _, v := range vi.finds {
								fmt.Fprintln(query.out.writer, colorstring.Color(string(v.Byte())))
							}
						}
						query.out.m.Unlock()
					}

					atomic.AddInt32(&send_in_count, 1)
					mutex.RUnlock()
				case send := <-f_reader_scan:
					mutex.Lock()
					// log.Println("send all", send)
					atomic.StoreInt32(&all, send)
					mutex.Unlock()
				}

				if send_in_count == all {
					// log.Println("scan finished================", finds_count, all)
					// log.Println("send_in_count:", send_in_count)

					c_scan_main <- finds_count
					atomic.StoreInt32(&send_in_count, 0)
					atomic.StoreInt32(&all, -1)
				}
			}
		}()
	}
}

type find struct {
	firstline int
	name      string
	rawcode   []byte
	comment   []byte
}

func (this *find) Byte_line() []byte {
	fields := bytes.FieldsFunc(this.rawcode, func(r rune) bool {
		return r == '\n'
	})

	var singleline bool
	if len(fields) == 2 {
		singleline = true
	} else {
		singleline = false
	}

	startline := this.firstline
	addlength := len(fields)*(len(strconv.Itoa(len(fields)+startline))+len(": ")) + 20
	bs := make([]byte, 0, len(this.rawcode)+len(this.comment)*3+addlength)

	ptr := unsafe.Pointer(&bs)

	//add filedir
	bs = append(bs, []byte("[yellow]")...)
	bs = append(bs, fields[0]...)
	bs = append(bs, byte('\n'))
	//add comment
	if this.comment != nil {
		bs = append(bs, this.comment...)
		bs = append(bs, byte('\n'))
	}

	//add the code
	if singleline {
		bs = append(bs, []byte("[cyan]"+strconv.Itoa(startline)+": [green]")...)
		bs = append(bs, fields[1]...)
	} else {
		for _, field := range fields[1:] {
			bs = append(bs, []byte("[cyan]"+strconv.Itoa(startline)+": [green]")...)
			bs = append(bs, field...)
			bs = append(bs, byte('\n'))
			startline++
		}
	}

	if unsafe.Pointer(&bs) != ptr {
		panic("the cache is not enought")
	}
	return bs
}

func (this *find) Byte() []byte {
	bs := make([]byte, 0, len(this.rawcode)+len(this.comment))
	bs = append(bs, this.rawcode...)
	if this.comment != nil {
		bs = append(bs, this.comment...)
	}

	return bs
}

//for sort
type Finds []*find

func (this Finds) Len() int {
	return len(this)
}

func (this Finds) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this Finds) Less(i, j int) bool {
	return this[i].name < this[j].name
}

//visitor
type visitor struct {
	astfile *ast.File
	fset    *token.FileSet
	raw     []byte

	filename string
	finds    []*find
}

func (this *visitor) add(c *find) {
	this.finds = append(this.finds, c)
}

func new_visitor(filename string, data []byte) *visitor {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, data, parser.ParseComments)
	if err != nil {
		//log.Println("is here===========", err)
		return nil
	}

	v := &visitor{
		filename: filename,
		astfile:  f,
		raw:      data,
		fset:     fset,
	}
	v.inspect()

	return v
}

func (this *visitor) inspect() {
	ast.Inspect(this.astfile, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		switch n := node.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					this.type_spec(n, s)
					continue
				case *ast.ValueSpec:
					this.value_spec(n, s)
					continue
				}
			}
			return false
		case *ast.FuncDecl:
			this.func_decl(n)
			return false
		}
		return true
	})
}

func (this *visitor) value_spec(n *ast.GenDecl, s *ast.ValueSpec) {
	var index int = -1
	for k, v := range s.Names {
		if query.name != v.Name {
			continue
		}
		index = k
	}
	if index == -1 {
		return
	}

	if this.finds == nil {
		this.finds = make([]*find, 0, 4)
	}

	firstline := this.fset.Position(s.Names[index].Pos()).Line

	var slice = make([]byte, 0, 1024*2)
	slice = append(slice, []byte("// ")...)
	slice = append(slice, []byte(this.filename)...)
	slice = append(slice, byte('\n'))
	var doc []byte

	doc = getComments(s.Doc)
	if doc != nil {
		slice = append(slice, doc...)
		slice = append(slice, byte('\n'))
	}

	//code
	slice = append(slice, []byte(n.Tok.String())...)
	slice = append(slice, byte(' '))
	slice = append(slice, []byte(query.name)...)

	if s.Type != nil {
		slice = append(slice, byte(' '))
		slice = append(slice, this.raw[s.Type.Pos()-1:s.Type.End()]...)
	}

	if s.Values != nil {
		slice = append(slice, []byte(" = ")...)
		slice = append(slice, this.raw[s.Values[index].Pos()-1:s.Values[index].End()]...)
	}

	if s.Comment != nil {
		slice = append(slice, this.raw[s.Comment.Pos()-1:s.Comment.End()]...)
	}

	comment := getComments(n.Doc)

	this.add(&find{firstline, query.name, slice, comment})
}

func (this *visitor) type_spec(n *ast.GenDecl, s *ast.TypeSpec) {
	var name string
	switch {
	case query.name != "" && query.name == s.Name.Name:
		name = query.name
	case query.name == "" && query.stru == s.Name.Name:
		//just for make the one to in the first num
		name = string(rune(12)) //12 < '_' , in the legal variable name("_0-9a-zA-Z")
	default:
		return
	}

	if this.finds == nil {
		this.finds = make([]*find, 0, 4)
	}

	firstline := this.fset.Position(s.Name.Pos()).Line
	slice := make([]byte, 0, len("type  //")+int(s.Type.End()-s.Name.Pos()-1)+len(this.filename)+20)

	slice = append(slice, []byte("// ")...)
	slice = append(slice, []byte(this.filename)...)
	slice = append(slice, byte('\n'))

	slice = append(slice, []byte("type ")...)
	slice = append(slice, this.raw[s.Name.Pos()-1:s.Type.End()]...)

	this.add(&find{firstline, name, slice, getComments(n.Doc)})
}

func (this *visitor) func_get_reciver(n *ast.FuncDecl) bool {
	if query.stru == "" {
		return true
	}
	if n.Recv == nil {
		return false
	}

	for _, field := range n.Recv.List {
		if query.stru == parseExpr(field.Type) {
			return true
		}
	}
	return false
}

func (this *visitor) func_decl(n *ast.FuncDecl) {
	//new
	switch {
	case !this.func_get_reciver(n):
		return
	case query.name != "" && query.name != n.Name.Name:
		return
	}

	if this.finds == nil {
		this.finds = make([]*find, 0, 4)
	}

	firstline := this.fset.Position(n.Type.Func).Line

	//new
	var name string = n.Name.Name

	start := n.Type.Func
	end := n.Type.End()
	if n.Body != nil {
		end = n.Body.End()
	}
	slice := make([]byte, 0, int(end-start)+len(this.filename)+20)

	slice = append(slice, []byte("// ")...)
	slice = append(slice, []byte(this.filename)...)
	slice = append(slice, byte('\n'))
	slice = append(slice, this.raw[start-1:end]...)

	this.add(&find{firstline, name, slice, getComments(n.Doc)})
}

//util
func getComments(n *ast.CommentGroup) []byte {
	if n == nil {
		return nil
	}
	slice := make([]string, len(n.List))

	for k, v := range n.List {
		slice[k] = v.Text
	}

	return []byte(strings.Join(slice, "\n"))
}

func parseExpr(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.StarExpr:
		return parseExpr(n.X)
	case *ast.Ident:
		return n.Name
	default:
		log.Fatalln(reflect.TypeOf(expr).String())
		return ""
	}
	return ""
}
