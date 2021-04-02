package pchi

import (
	"fmt"
	"net/http"
	regexp2 "regexp"
	"sort"
	"strings"
)

type NodeType int

const (
	static   NodeType = iota // 静态链接: /hello/check_heath
	param                    // 带参数的链接： /user/{name}
	regexp                   // 正则表达式的链接 /user/{id:[0-9]+}
	wildcard                 // 带 * 的链接 /user/*
)

type EndPoint struct {
	Pattern    string
	Handler    http.Handler
	MethodType HttpMethodType
}

type nodes []*Node

func (n nodes) Len() int {
	return len(n)
}

func (n nodes) Less(i, j int) bool {
	return n[i].Prefix < n[j].Prefix
}

func (n nodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

type Node struct {
	NodeType     NodeType
	Prefix       string
	Child        [wildcard + 1]nodes
	EndPoints    []*EndPoint
	ParamKey     string
	Express      string
	RegexpMethod *regexp2.Regexp
}

func (node *Node) InsertRouter(pattern string, method HttpMethodType, handler http.Handler) *Node {
	parent := node
	search := pattern

	for {
		var bi, be int
		var find bool
		nodeType, es, ed, key, express := ParsePattern(search)
		if search[0] == '{' || search[0] == '*' {
			be = ed
		} else {
			be = es - 1
			if es == 0 {
				be = ed
			}
			nodeType = static
		}

		prefix := search[bi : be+1]
		child := parent.GetChild(nodeType, prefix, key, express)
		if child == nil {
			child = &Node{
				NodeType: nodeType,
				Prefix:   prefix,
			}
			if nodeType > static {
				child.ParamKey = key
				child.Express = express
			}
			if child.Express != "" {
				rm, err := regexp2.Compile(child.Express)
				if err != nil {
					panic(fmt.Sprintf("pchi:编译正则表达式失败，express = %s, pattern = %s, method = %s, err = %s", child.Express, pattern, HttpMethodString[method], err.Error()))
				}
				child.RegexpMethod = rm
			}
			parent.AddChild(child)
			find = true
		}

		// { 或者 * 开头，继续查找后面的部分
		if find || nodeType != static {
			if be == len(search)-1 {
				child.AddEndPoint(method, pattern, handler)
				return child
			}
			parent = child
			search = search[be+1:]
			continue
		}

		pi := patternEqLen(child.Prefix, prefix)

		// 1、child.Prefix 为 /ab prefix 为 /abc
		// 2、child.Prefix 为 /ab prefix 为 /ab
		if pi == len(child.Prefix) {
			if es == 0 && child.Prefix == prefix {
				child.AddEndPoint(method, pattern, handler)
				return child
			}
			parent = child
			search = search[bi+pi:]
			continue
		}

		// 1、child.Prefix 为 /ab，prefix 为 /ac
		// 2、child.Prefix 为 /ab，prefix 为 /a
		oldChild := *child
		child.Clean()
		child.Prefix = prefix[:pi]
		oldChild.Prefix = oldChild.Prefix[pi:]
		child.AddChild(&oldChild)

		if pi < len(prefix) {
			newChild := &Node{
				NodeType: static,
				Prefix:   prefix[pi:],
			}
			child.AddChild(newChild)
			child = newChild
		}
		// search 为 static
		if es == 0 {
			child.AddEndPoint(method, pattern, handler)
			return child
		}

		parent = child
		search = search[be+1:]
	}
}

func (node *Node) InsertNode(child *Node) {
	if len(child.EndPoints) > 0 {
		for _, ed := range child.EndPoints {
			node.InsertRouter(ed.Pattern, ed.MethodType, ed.Handler)
		}
		return
	}

	parent := node.InsertRouter(child.Prefix, Sub, nil)
	for _, nodes := range child.Child {
		for _, cn := range nodes {
			parent.InsertNode(cn)
		}
	}
}

func (node *Node) FindNode(context *RouterContext, pattern string) *Node {
	parent := node
	search := pattern

	for {
		child := parent.findChild(search)
		if child == nil {
			return child
		}
		switch child.NodeType {
		case param, regexp:
			index := strings.IndexByte(search, '/')
			if index == -1 {
				index = len(search)
			}
			context.ParamKey = append(context.ParamKey, child.ParamKey)
			context.ParamValue = append(context.ParamValue, search[:index])
			if index == len(search) {
				return child
			}
			parent = child
			search = search[index:]
		case wildcard:
			return child
		case static:
			if search == child.Prefix {
				return child
			}
			pi := patternEqLen(child.Prefix, search)
			parent = child
			search = search[pi:]
		}
	}
}

func (node *Node) FindHandler(context *RouterContext, pattern string, method HttpMethodType) http.Handler {
	child := node.FindNode(context, pattern)
	if child == nil {
		return nil
	}
	for _, ep := range child.EndPoints {
		if ep.MethodType&method != 0 {
			return ep.Handler
		}
	}
	return nil
}

func (node *Node) findChild(pattern string) *Node {
	for _, nodes := range node.Child {
		for _, node := range nodes {
			switch node.NodeType {
			case static:
				if pattern[0] == node.Prefix[0] {
					return node
				}
			case param, wildcard:
				return node
			case regexp:
				index := strings.IndexByte(pattern, '/')
				if index == -1 {
					index = len(pattern)
				}
				if node.RegexpMethod.MatchString(pattern[:index]) {
					return node
				}

			}
		}
	}
	return nil
}

func (node *Node) AddChild(child *Node) {
	node.Child[child.NodeType] = append(node.Child[child.NodeType], child)
	sort.Sort(node.Child[child.NodeType])
}

func (node *Node) GetChild(nodeType NodeType, pattern, key, reg string) *Node {
	for _, child := range node.Child[nodeType] {
		switch nodeType {
		case static:
			if child.Prefix[0] == pattern[0] {
				return child
			}
		case param:
			if child.ParamKey == key {
				return child
			}
		case regexp:
			if child.ParamKey == key && child.Express == reg {
				return child
			}
		case wildcard:
			return child
		}
	}

	return nil
}

func ParsePattern(pattern string) (NodeType, int, int, string, string) {
	pl := len(pattern)
	expressStart := strings.IndexByte(pattern, '{')
	wildcardStart := strings.IndexByte(pattern, '*')

	if wildcardStart >= 0 {
		if wildcardStart != len(pattern)-1 {
			panic(fmt.Sprintf("pchi: 通配符 * 必须出现在链接末尾, pattern = %s", pattern))
		}
		return wildcard, wildcardStart, wildcardStart + 1, "", ""
	}

	if expressStart >= 0 {
		nodeType := param
		var key, reg string
		expressCount := 1
		var expressEnd int
		for i := expressStart + 1; i < pl; i++ {
			if pattern[i] == '{' {
				expressCount++
			} else if pattern[i] == '}' {
				expressCount--
			}
			if expressCount == 0 {
				expressEnd = i
				break
			}
		}
		key = pattern[expressStart+1 : expressEnd]
		if params := strings.Split(key, ":"); len(params) > 1 {
			key = params[0]
			reg = params[1]
			// 强制全部匹配
			if reg[0] != '^' {
				reg = "^" + reg
			}
			if reg[len(reg)-1] != '$' {
				reg = reg + "$"
			}
			nodeType = regexp
		}
		return nodeType, expressStart, expressEnd, key, reg
	}

	return static, 0, pl - 1, "", ""

}

func (node *Node) Clean() {
	node.Prefix = ""
	node.EndPoints = node.EndPoints[:0]
	node.Express = ""
	node.ParamKey = ""
	node.Child = [4]nodes{}
}

func (node *Node) AddEndPoint(method HttpMethodType, pattern string, handler http.Handler) {
	for _, ep := range node.EndPoints {
		if ep.MethodType&method != 0 {
			panic(fmt.Sprintf("pchi: 已经存在方法 %s 对应的 handler 了，pattern = %s", HttpMethodString[method], pattern))
		}
	}
	if method == Sub {
		return
	}
	nep := &EndPoint{Pattern: pattern, Handler: handler, MethodType: method}
	node.EndPoints = append(node.EndPoints, nep)
}

func (node *Node) GetEndPoint(method HttpMethodType) *EndPoint {
	for _, ep := range node.EndPoints {
		if ep.MethodType&method != 0 {
			return ep
		}
	}

	return nil
}

func patternEqLen(s1, s2 string) int {
	for i := 0; i < len(s1); i++ {
		if i == len(s2) {
			return i
		}
		if s1[i] != s2[i] {
			return i
		}
	}
	return len(s1)
}
