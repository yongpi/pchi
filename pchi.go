package pchi

import (
	"context"
	"net/http"
)

type HttpMethodType int

const (
	Get HttpMethodType = 1 << iota
	Head
	Post
	Put
	Patch
	Delete
	Connect
	Options
	Trace
)

var RouterContextKey = &contextKey{name: "RouterContextKey"}

const AllMethod = Get | Head | Post | Put | Patch | Delete | Connect | Options | Trace

var HttpMethodMap = map[string]HttpMethodType{
	http.MethodGet:     Get,
	http.MethodHead:    Head,
	http.MethodPost:    Post,
	http.MethodPut:     Put,
	http.MethodPatch:   Patch,
	http.MethodDelete:  Delete,
	http.MethodConnect: Connect,
	http.MethodOptions: Options,
	http.MethodTrace:   Trace,
}

var HttpMethodString = map[HttpMethodType]string{
	Get:     http.MethodGet,
	Head:    http.MethodHead,
	Post:    http.MethodPost,
	Put:     http.MethodPut,
	Patch:   http.MethodPatch,
	Delete:  http.MethodDelete,
	Connect: http.MethodConnect,
	Options: http.MethodOptions,
	Trace:   http.MethodTrace,
}

type MiddleWare func(next http.Handler) http.Handler

// 过滤器
type HttpFilter struct {
	MiddleWare
	// 需要过滤的 router 列表, url 后面跟 method,可以组合 method，当 method 为 all 时，支持所有 method
	// 例如：[]string{"/a/s/{sku_id}:get", "/c/n:post&get", "/b/c:all"}
	Routers []string
}

type RouterContext struct {
	ParamKey   []string
	ParamValue []string
}

func (context *RouterContext) Clean() {
	context.ParamValue = context.ParamValue[:0]
	context.ParamKey = context.ParamKey[:0]
}

type contextKey struct {
	name string
}

func GetURLParam(context context.Context, key string) string {
	return GetURLParamByIndex(context, key, 1)
}

func GetURLParamByIndex(context context.Context, key string, index int) string {
	routerContext, ok := context.Value(RouterContextKey).(*RouterContext)
	if !ok {
		return ""
	}
	var keyIndex int
	for paramIndex, paramKey := range routerContext.ParamKey {
		if paramKey == key {
			keyIndex++
		}
		if keyIndex == index {
			return routerContext.ParamValue[paramIndex]
		}
	}

	return ""
}

type HttpRouter interface {
	http.Handler
	RouterHandler(pattern string, methodType HttpMethodType, handler http.Handler)
	Middleware(middleware MiddleWare)
	Get(pattern string, handler http.Handler)
	Post(pattern string, handler http.Handler)
	Put(pattern string, handler http.Handler)
	Delete(pattern string, handler http.Handler)
	Patch(pattern string, handler http.Handler)
	Module(pattern string, fn func(r HttpRouter))
	Filter(filter HttpFilter)
}
