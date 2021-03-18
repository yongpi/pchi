package pchi

import (
	"net/http"
	"testing"
)

func TestRouter(t *testing.T) {
	root := &Node{}
	helloFun := func(w http.ResponseWriter, r *http.Request) {

	}
	root.InsertNode("/hello", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/ha", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/c", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/c/{name}", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/hell/{id:[1-9]+}", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/{name}/1111/hello/{aaa}", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/{name}/2222", Get, http.HandlerFunc(helloFun))
	root.InsertNode("/darwin/{name}/{id:[1-9]+}/aaa", Get, http.HandlerFunc(helloFun))

	urlCheck := map[string]RouterContext{
		"/hello":                {},
		"/c/aaa":                {ParamKey: []string{"name"}, ParamValue: []string{"aaa"}},
		"/hell/123":             {ParamKey: []string{"id"}, ParamValue: []string{"123"}},
		"/sku/1111/hello/ccc":   {ParamKey: []string{"name", "aaa"}, ParamValue: []string{"sku", "ccc"}},
		"/darwin/sku/11111/aaa": {ParamKey: []string{"name", "id"}, ParamValue: []string{"sku", "11111"}}}
	for url, param := range urlCheck {
		context := &RouterContext{}
		n := root.FindNode(context, "/hello")
		if n == nil {
			t.Errorf("找不到对应的路由， url = %s", url)
		} else {
			for key, value := range context.ParamKey {
				if value != param.ParamKey[key] {
					t.Errorf("参数 key 获取失败，url = %s, param key = %v, context key = %v", url, param.ParamKey, context.ParamKey)
				}
			}

			for key, value := range context.ParamValue {
				if value != param.ParamValue[key] {
					t.Errorf("参数 value 获取失败，url = %s, param value = %v, context value = %v", url, param.ParamValue, context.ParamValue)
				}
			}
		}
	}

}
