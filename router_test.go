package pchi

import (
	"fmt"
	"log"
	"net/http"
	"testing"
)

func TestNewHttpRouter(t *testing.T) {
	router := NewHttpRouter()
	router.Middleware(func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("middle ware")
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})

	router.Filter(HttpFilter{MiddleWare: func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("filter")
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}, Routers: []string{"/a:get"}})

	router.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("get /a")
		_, _ = w.Write([]byte("get /a"))
	}))

	router.Module("/s", func(r HttpRouter) {
		r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("get /s")
			_, _ = w.Write([]byte("get /a"))
		}))
		r.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("get /s/a")
			_, _ = w.Write([]byte("get /a"))
		}))
	})

	fmt.Println("listen http")
	err := http.ListenAndServe("127.0.0.1:1926", router)
	if err != nil {
		log.Fatal(err)
	}
}
