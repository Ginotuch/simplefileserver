package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/Ginotuch/simplefileserver/backend"
	auth "github.com/abbot/go-http-auth"
)

func main() {
	rootDir := flag.String("root", "./temp", "The root directory for the hosted files.")
	port := flag.String("port", "8090", "Port number to listen on.")
	cert := flag.String("cert", "./localhost.crt", "Cert file for TLS")
	key := flag.String("key", "./localhost.key", "Key file for TLS")
	flag.Parse()

	fmt.Printf("registered root dir \"%s\"\n", *rootDir)

	newServer := backend.NewServer(*rootDir, backend.LogDebug)

	authenticator := auth.NewBasicAuthenticator("simplefileserver", auth.HtdigestFileProvider(".htdigest"))

	mux := http.NewServeMux()

	mux.HandleFunc("/download/", authenticator.Wrap(newServer.Download))
	mux.HandleFunc("/walk/", authenticator.Wrap(newServer.Walk))
	mux.HandleFunc("/favicon.ico", authenticator.Wrap(newServer.Favicon))
	mux.HandleFunc("/", newServer.Home)

	log.Fatal(http.ListenAndServeTLS(":"+*port, *cert, *key, mux))
}
