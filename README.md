项目介绍
======

背景：

以前经常需要查询源代码，可是在windows上面要经过好多的路径才能看，而mac更加麻烦，所以想到写一个查找func的功能函数。


目前已经实现的功能：
======

1.目前已实现查找标准库、安装在第一个GOPATH内第三方包的函数，并且已经可以显示函数注释了。
2.可查找接口、结构及注释。
3.实现查找结果向文件输出

发现模糊查询非本功能必须，如有必要实现其实也简单；功能实现非常的不优雅，抱歉之至啊!

尚不完善，可能有bug，尽请谅解

	go get -u github.com/eruca/gofn

go get 后应该已经安装好了，可以使用了，使用例子如下：

	gofn strings.Index
	gofn bufio.Reader
	gofn bufio.SplitFunc
	gofn beego.Input

下面两个功能的实现，非常难看，本想用flag，可是不太懂，而且要求太多，比如 gofn string.Index -out实现默认输出，可是发现 -out 非第二个参数无法执行，请高手指点！

	gofn strings.Index out //默认输出到 1st GOPATH/src/gofn/result.go
	gofn strings.Index out d:\\test.go //目标输出

希望：
==
1.查看github.com内高手使用的示例
2.性能改善



