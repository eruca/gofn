项目介绍
======

背景：

以前经常需要查询源代码，可是在windows上面要经过好多的路径才能看，而mac更加麻烦，所以想到写一个查找func的功能函数。


目前已经实现的功能：
======

1.目前已实现查找标准库、安装在第一个GOPATH内第三方包的函数，并且已经可以显示函数注释了。
2.可查找接口、结构及注释。


尚不完善，可能有bug，尽请谅解

	go get github.com/eruca/gofn

现在已经拆分为2个文件，所以命令行转到eruca文件夹下，执行 go install gofn

	gofn strings.Index
	gofn bufio.Reader
	gofn bufio.SplitFunc
	gofn beego.Input

希望：
==
1.实现查找结果向文件输出
2.模糊查询
3.查看github.com内高手使用的示例



