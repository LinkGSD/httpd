package main

import (
	"fmt"
	"httpd/httpd"
	"io"
	"io/ioutil"
	"os"
)

type myHandler struct{}

func (*myHandler) ServeHTTP(w httpd.ResponseWriter, r *httpd.Request) {
	if r.URL.Path == "/photo" {
		file, err := os.Open("test.webp")
		if err != nil {
			fmt.Println("open file error:", err)
			return
		}
		io.Copy(w, file)
		file.Close()
		return
	}
	data, err := ioutil.ReadFile("test.html")
	if err != nil {
		fmt.Println("readFile test.html error: ", err)
		return
	}
	w.Write(data)
}

func main() {
	svr := httpd.Server{
		Addr:    "127.0.0.1:8080",
		Handler: &myHandler{},
	}
	panic(any(svr.ListenAndServe()))
}
