package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/Ginotuch/simplefileserver/backend"
)

func main() {
	rootDir := flag.String("root", "./temp", "The root directory for the hosted files.")
	port := flag.String("port", "8090", "Port number to listen on.")
	cert := flag.String("cert", "./localhost.crt", "Cert file for TLS")
	key := flag.String("key", "./localhost.key", "Key file for TLS")
	flag.Parse()

	fmt.Printf("registered root dir \"%s\"\n", *rootDir)
	fmt.Printf("Starting server and listening on port: %s\n", *port)

	newServer := backend.NewServer(*rootDir, zap.DebugLevel)

	log.Fatal(http.ListenAndServeTLS(":"+*port, *cert, *key, newServer))
}
