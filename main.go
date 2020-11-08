package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/Ginotuch/simplefileserver/backend"
)

func main() {
	rootDir := flag.String("root", "./temp", "The root directory for the hosted files.")
	port := flag.String("port", "8090", "Port number to listen on.")
	cert := flag.String("cert", "./localhost.crt", "Cert file for TLS")
	key := flag.String("key", "./localhost.key", "Key file for TLS")
	expire := flag.Int("expire", 48, "Hours until temporary links expire")
	flag.Parse()

	for _, fileName := range []string{*cert, *key, ".htdigest"} {
		fileHandle, err := os.Open(fileName)
		if os.IsNotExist(err) || os.IsPermission(err) {
			log.Fatalf("%s not found or access to file denied\n", fileName)
		}
		err = fileHandle.Close()
		if err != nil {
			log.Fatalf("Unable to close file %s\nError: %s", fileName, err)
		}
	}

	fmt.Printf("registered root dir \"%s\"\n", *rootDir)
	fmt.Printf("Starting server and listening on port: %s\n", *port)

	newServer := backend.NewServer(*rootDir, zap.DebugLevel, *expire)

	log.Fatal(http.ListenAndServeTLS(":"+*port, *cert, *key, newServer))
}
