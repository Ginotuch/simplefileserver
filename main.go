package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Ginotuch/simplefileserver/backend"
	auth "github.com/abbot/go-http-auth"
)

func Secret(user, realm string) string {
	return "change me ;)"
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Requires single argument of root dir\n $ main.go /path/to/dir")
	}
	rootDir := os.Args[1]

	fmt.Printf("registered root dir \"%s\"\n", rootDir)

	newServer := backend.NewServer(rootDir, backend.LogDebug)

	authenticator := auth.NewBasicAuthenticator("example.com", Secret)

	mux := http.NewServeMux()

	mux.HandleFunc("/download/", authenticator.Wrap(newServer.Download))
	mux.HandleFunc("/walk/", authenticator.Wrap(newServer.Walk))
	mux.HandleFunc("/favicon.ico", authenticator.Wrap(newServer.Favicon))
	mux.HandleFunc("/", newServer.Home)

	_ = http.ListenAndServeTLS(":8090", "localhost.crt", "localhost.key", mux)
}
