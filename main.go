package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type Server interface {
	Hello(w http.ResponseWriter, req *http.Request)
	Headers(w http.ResponseWriter, req *http.Request)
	E404(w http.ResponseWriter, req *http.Request)
	Home(w http.ResponseWriter, req *http.Request)
	Walk(w http.ResponseWriter, req *http.Request)
}

type ServerStruct struct {
	rootDir string
}

func (s *ServerStruct) Hello(w http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(w, "Hello\n")
}

func (s *ServerStruct) Headers(w http.ResponseWriter, req *http.Request) {

	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func (s *ServerStruct) E404(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "my404\n")
}

func (s *ServerStruct) Home(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		s.E404(w, req)
		return
	}
	fmt.Fprintf(w, "Home\n")
}

func (s *ServerStruct) Walk(w http.ResponseWriter, req *http.Request) {
	requestedFolder := path.Join(strings.Split(req.URL.Path, "/")[2:]...)
	absPath := path.Join(s.rootDir, requestedFolder)

	fileInfo, err := os.Stat(absPath)
	if unix.Access(absPath, unix.R_OK) != nil || err != nil || !fileInfo.IsDir() {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Either the requested directory doesn't exist or access was denied")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	requestedFolder += "/"
	absPath += "/"
	pageHtml := fmt.Sprintf("<!DOCTYPE html><html><head><title>%s</title></head><body>", requestedFolder)

	pageHtml += fmt.Sprintf("<h1>Listing for dir: %s</h1>", requestedFolder)
	pageHtml += fmt.Sprintf("<ul>")
	files, err := ioutil.ReadDir(absPath)
	for _, f := range files {
		name := f.Name()
		if f.IsDir() {
			name += "/"
		}
		pageHtml += fmt.Sprintf("<li>%s</li>", name)
	}
	pageHtml += fmt.Sprintf("</ul></body></html>")
	_, err = fmt.Fprint(w, pageHtml)
	if err != nil {
		log.Fatal("Unable to write response")
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Requires single argument of root dir\n $ main.go /path/to/dir")
	}
	rootDir := os.Args[1]

	fmt.Printf("registered root dir \"%s\"\n", rootDir)

	var newServer Server = &ServerStruct{rootDir: rootDir}

	mux := http.NewServeMux()

	mux.HandleFunc("/walk/", newServer.Walk)
	mux.HandleFunc("/hello", newServer.Hello)
	mux.HandleFunc("/headers", newServer.Headers)
	mux.HandleFunc("/", newServer.Home)

	_ = http.ListenAndServe(":8090", mux)
}
