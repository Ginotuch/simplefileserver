package main

import (
	"fmt"
	backend "github.com/ginotuch/simplefileserver/backend"
	"log"
	"net/http"
	"os"

	_ "go.uber.org/zap"
)

func main() {
	cfg, err := backend.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	for _, fileName := range []string{cfg.CertFile, cfg.KeyFile} {
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) || os.IsPermission(err) {
			log.Fatalf("%s not found or access to file denied\n", fileName)
		}
	}

	fmt.Printf("registered root dir \"%s\"\n", cfg.RootDir)
	fmt.Printf("Starting server and listening on port: %s\n", cfg.Port)

	server, err := backend.NewServer(cfg)
	if err != nil {
		log.Fatalf("Unable to create server: %v", err)
	}

	log.Fatal(http.ListenAndServeTLS(":"+cfg.Port, cfg.CertFile, cfg.KeyFile, server))
}
