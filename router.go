package pchi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var _ HttpRouter = &PHttpRouter{}

func NewHttpRouter() *PHttpRouter {
	router := &PHttpRouter{
		root: &Node{},
	}
	router.contextPool.New = func() interface{} {
		return &RouterContext{}
	}

	return router
}

type HandlerFilter struct {
	HttpFilter
	Method HttpMethodType
}

type PHttpRouter struct {
	contextPool sync.Pool
	handler     http.Handler
	middleWares []MiddleWare
	root        *Node
	filters     map[string][]HandlerFilter
}

func (router *PHttpRouter) Filter(filter HttpFilter) {
	if router.filters == nil {
		router.filters = make(map[string][]HandlerFilter)
	}

	for _, routerPattern := range filter.Routers {
		sp := strings.Split(routerPattern, ":")
		method := strings.ToUpper(sp[len(sp)-1])
		var httpMethod HttpMethodType
		if method == "ALL" {
			httpMethod = AllMethod
		} else {
			methods := strings.Split(method, "&")
			for _, strMethod := range methods {
				hm, ok := HttpMethodMap[strings.ToUpper(strMethod)]
				if !ok {
					panic(fmt.Sprintf("pchi: filter routers 中含有的 method 非法，pattern = %s, routers = %v", routerPattern, filter.Routers))
				}
				httpMethod |= hm
			}
		}

		pattern := routerPattern[:len(routerPattern)-len(method)-1]
		router.filters[pattern] = append(router.filters[pattern], HandlerFilter{HttpFilter: filter, Method: httpMethod})
	}
}

// 适用于子集 urls ，子集中如果有 middleware 则只有子集里的 Handler 会使用
func (router *PHttpRouter) Module(pattern string, fn func(r HttpRouter)) {
	r := NewHttpRouter()
	fn(r)
	if r.filters != nil {
		panic(fmt.Sprintf("pchi: 子 router 不允许添加 filter"))
	}

	router.root.InsertNode(pattern, r.root, r.middleWares)
}

func (router *PHttpRouter) Get(pattern string, handler http.Handler) {
	router.RouterHandler(pattern, Get, handler)
}

func (router *PHttpRouter) Post(pattern string, handler http.Handler) {
	router.RouterHandler(pattern, Post, handler)
}

func (router *PHttpRouter) Put(pattern string, handler http.Handler) {
	router.RouterHandler(pattern, Put, handler)
}

func (router *PHttpRouter) Delete(pattern string, handler http.Handler) {
	router.RouterHandler(pattern, Delete, handler)
}

func (router *PHttpRouter) Patch(pattern string, handler http.Handler) {
	router.RouterHandler(pattern, Patch, handler)
}

func (router *PHttpRouter) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	router.handler.ServeHTTP(response, request)
}

func (router *PHttpRouter) RouterHandler(pattern string, methodType HttpMethodType, handler http.Handler) {
	if router.handler == nil {
		router.buildBaseHandler()
	}
	// 填充 filter
	filters, ok := router.filters[pattern]
	var fm []MiddleWare
	if ok {
		for _, filter := range filters {
			if filter.Method&methodType != 0 {
				fm = append(fm, filter.MiddleWare)
			}
		}
	}
	handler = linkHandler(fm, handler)
	router.root.InsertRouter(pattern, methodType, handler)
}

func (router *PHttpRouter) Middleware(middleware MiddleWare) {
	router.middleWares = append(router.middleWares, middleware)
}

func (router *PHttpRouter) buildBaseHandler() {
	if router.handler != nil {
		return
	}
	handler := http.Handler(http.HandlerFunc(router.routerHttp))
	router.handler = linkHandler(router.middleWares, handler)
}

func (router *PHttpRouter) routerHttp(response http.ResponseWriter, request *http.Request) {
	pattern := request.URL.RawPath
	if pattern == "" {
		pattern = request.URL.Path
	}
	httpMethod := HttpMethodMap[request.Method]
	routerContext := router.contextPool.Get().(*RouterContext)
	routerContext.Clean()

	node := router.root.FindNode(routerContext, pattern)
	if node == nil {
		http.NotFound(response, request)
		return
	}
	endPoint := node.GetEndPoint(httpMethod)
	if endPoint == nil {
		http.Error(response, fmt.Sprintf("pchi: %s 的 %s 方法下的 handler 不存在", pattern, request.Method), 404)
		return
	}
	request = request.WithContext(context.WithValue(request.Context(), RouterContextKey, routerContext))
	endPoint.Handler.ServeHTTP(response, request)

	routerContext.Clean()
	router.contextPool.Put(routerContext)
	return
}

func linkHandler(fns []MiddleWare, handler http.Handler) http.Handler {
	if len(fns) == 0 {
		return handler
	}
	handler = fns[len(fns)-1](handler)
	for i := len(fns) - 2; i >= 0; i-- {
		handler = fns[i](handler)
	}
	return handler
}
