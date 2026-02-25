package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	clientpkg "github.com/j178/reverse-tunnel/internal/client"
	serverpkg "github.com/j178/reverse-tunnel/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "server":
		runServer(os.Args[2:])
	case "client":
		runClient(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	listenPublic := fs.String("listen-a", ":18080", "public listener address")
	listenClient := fs.String("listen-b", ":19090", "client listener address")
	pairTimeout := fs.Duration("pair-timeout", 10*time.Second, "max wait for client data connection")
	fs.Parse(args)

	service := serverpkg.New(serverpkg.Config{
		ListenPublic: *listenPublic,
		ListenClient: *listenClient,
		PairTimeout:  *pairTimeout,
	})

	if err := service.Run(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

func runClient(args []string) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	serverAddr := fs.String("server", "127.0.0.1:19090", "server client-port address")
	targetAddr := fs.String("target", "127.0.0.1:8080", "local target service address")
	reconnectBackoff := fs.Duration("reconnect", 2*time.Second, "reconnect backoff when control channel drops")
	fs.Parse(args)

	if *targetAddr == "" {
		log.Fatalf("target cannot be empty")
	}

	agent := clientpkg.New(clientpkg.Config{
		ServerAddr:       *serverAddr,
		TargetAddr:       *targetAddr,
		ReconnectBackoff: *reconnectBackoff,
	})

	agent.Run()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  reverse-tunnel <subcommand> [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "  server    Start reverse tunnel server\n")
	fmt.Fprintf(os.Stderr, "  client    Start reverse tunnel client\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  reverse-tunnel server --listen-a :18080 --listen-b :19090\n")
	fmt.Fprintf(os.Stderr, "  reverse-tunnel client --server 127.0.0.1:19090 --target 127.0.0.1:8080\n")
}
