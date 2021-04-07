package pchi

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
)

func newHttpServer() {
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
	}, Routers: []string{"/a:get&post&put"}})

	router.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("get /a")
		_, _ = w.Write([]byte("get /a"))
	}))

	router.Module("/s", func(r HttpRouter) {
		r.Middleware(func(next http.Handler) http.Handler {
			fn := func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("module middle ware")
				next.ServeHTTP(w, r)
			}
			return http.HandlerFunc(fn)
		})
		r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("get /s")
			_, _ = w.Write([]byte("get /a"))
		}))
		r.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("get /s/a")
			_, _ = w.Write([]byte("get /a"))
		}))
	})

	err := http.ListenAndServe("127.0.0.1:1926", router)
	if err != nil {
		log.Fatal(err)
	}
}
func TestNewHttpRouter(t *testing.T) {
	go func() {
		newHttpServer()
	}()

	time.Sleep(50 * time.Millisecond)

	url := "http://127.0.0.1:1926/a"
	resp, err := http.Get(url)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get %s fail, http code = %d", url, resp.StatusCode)
	}

	url = "http://127.0.0.1:1926/s"
	resp, err = http.Get(url)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get %s fail, http code = %d", url, resp.StatusCode)
	}

	url = "http://127.0.0.1:1926/s/a"
	resp, err = http.Get(url)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get %s fail, http code = %d", url, resp.StatusCode)
	}

}
