package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Ginotuch/simplefileserver/backend"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Requires single argument of root dir\n $ main.go /path/to/dir")
	}
	rootDir := os.Args[1]

	fmt.Printf("registered root dir \"%s\"\n", rootDir)

	newServer := backend.NewServer(rootDir)

	mux := http.NewServeMux()

	mux.HandleFunc("/download/", newServer.Download)
	mux.HandleFunc("/walk/", newServer.Walk)
	mux.HandleFunc("/favicon.ico", newServer.Favicon)
	mux.HandleFunc("/", newServer.Home)

	_ = http.ListenAndServe(":8090", mux)
}
