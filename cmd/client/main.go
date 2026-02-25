package main

import (
	"flag"
	"log"
	"time"

	"tmp-proxy/internal/client"
)

func main() {
	serverAddr := flag.String("server", "127.0.0.1:19090", "server client-port address")
	targetAddr := flag.String("target", "127.0.0.1:8080", "local target service address")
	reconnectBackoff := flag.Duration("reconnect", 2*time.Second, "reconnect backoff when control channel drops")
	flag.Parse()

	if *targetAddr == "" {
		log.Fatalf("target cannot be empty")
	}

	agent := client.New(client.Config{
		ServerAddr:       *serverAddr,
		TargetAddr:       *targetAddr,
		ReconnectBackoff: *reconnectBackoff,
	})

	agent.Run()
}
