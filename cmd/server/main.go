package main

import (
	"flag"
	"log"
	"time"

	"tmp-proxy/internal/server"
)

func main() {
	listenPublic := flag.String("listen-a", ":18080", "public listener address")
	listenClient := flag.String("listen-b", ":19090", "client listener address")
	pairTimeout := flag.Duration("pair-timeout", 10*time.Second, "max wait for client data connection")
	flag.Parse()

	service := server.New(server.Config{
		ListenPublic: *listenPublic,
		ListenClient: *listenClient,
		PairTimeout:  *pairTimeout,
	})

	if err := service.Run(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
