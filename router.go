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
		Root: &Node{},
	}
	router.ContextPool.New = func() interface{} {
		return &RouterContext{}
	}

	return router
}

type PHttpRouter struct {
	handler     http.Handler
	middleWares []MiddleWare
	Root        *Node
	ContextPool sync.Pool
}

func (router *PHttpRouter) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	router.handler.ServeHTTP(response, request)
}

func (router *PHttpRouter) RouterHandler(pattern string, methodType HttpMethodType, handler http.Handler) {
	if router.handler == nil {
		router.buildBaseHandler()
	}
	router.Root.InsertNode(pattern, methodType, handler)
}

func (router *PHttpRouter) Middleware(middleware MiddleWare) {
	router.middleWares = append(router.middleWares, middleware)
}

func (router *PHttpRouter) buildBaseHandler() {
	if router.handler != nil {
		return
	}
	fn := http.HandlerFunc(router.routerHttp)
	handler := router.middleWares[len(router.middleWares)-1](fn)
	for i := len(router.middleWares) - 2; i >= 0; i-- {
		handler = router.middleWares[i](handler)
	}
	router.handler = handler
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
	endPoint, ok := node.EndPoints[httpMethod]
	if !ok {
		http.Error(response, fmt.Sprintf("pchi: %s 的 %s 方法下的 handler 不存在", pattern, request.Method), 404)
		return
	}
	request = request.WithContext(context.WithValue(request.Context(), RouterContextKey, routerContext))
	endPoint.Handler.ServeHTTP(response, request)

	routerContext.Clean()
	router.ContextPool.Put(routerContext)
	return

}
