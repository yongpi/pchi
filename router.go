package pchi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

var _ HttpRouter = &PHttpRouter{}

func NewHttpRouter() *PHttpRouter {
	router := &PHttpRouter{
		root: &Node{},
	}
	router.ContextPool.New = func() interface{} {
		return &RouterContext{}
	}

	return router
}

type PHttpRouter struct {
	ContextPool sync.Pool
	Handler     http.Handler
	MiddleWares []MiddleWare
	Root        *Node
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
	router.Handler.ServeHTTP(response, request)
}

func (router *PHttpRouter) RouterHandler(pattern string, methodType HttpMethodType, handler http.Handler) {
	if router.Handler == nil {
		router.buildBaseHandler()
	}
	router.Root.InsertNode(pattern, methodType, handler)
}

func (router *PHttpRouter) Middleware(middleware MiddleWare) {
	router.MiddleWares = append(router.MiddleWares, middleware)
}

func (router *PHttpRouter) buildBaseHandler() {
	if router.Handler != nil {
		return
	}
	fn := http.HandlerFunc(router.routerHttp)
	handler := router.MiddleWares[len(router.MiddleWares)-1](fn)
	for i := len(router.MiddleWares) - 2; i >= 0; i-- {
		handler = router.MiddleWares[i](handler)
	}
	router.Handler = handler
}

func (router *PHttpRouter) routerHttp(response http.ResponseWriter, request *http.Request) {
	pattern := request.URL.RawPath
	if pattern == "" {
		pattern = request.URL.Path
	}
	httpMethod := HttpMethodMap[request.Method]
	routerContext := router.ContextPool.Get().(*RouterContext)
	routerContext.Clean()

	node := router.Root.FindNode(routerContext, pattern)
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
	router.ContextPool.Put(routerContext)
	return
}
