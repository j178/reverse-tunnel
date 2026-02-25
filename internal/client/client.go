package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/j178/reverse-tunnel/internal/protocol"
	"github.com/j178/reverse-tunnel/internal/transport"
)

type Config struct {
	ServerAddr       string
	TargetAddr       string
	ReconnectBackoff time.Duration
}

type Client struct {
	config Config
}

func New(config Config) *Client {
	return &Client{config: config}
}

func (client *Client) Run() {
	for {
		err := client.runControlSession()
		if err != nil {
			log.Printf("control session ended: %v", err)
		}

		time.Sleep(client.config.ReconnectBackoff)
	}
}

func (client *Client) runControlSession() error {
	conn, err := net.Dial("tcp", client.config.ServerAddr)
	if err != nil {
		return fmt.Errorf("dial server: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(protocol.BuildControlHello())); err != nil {
		return fmt.Errorf("send control hello: %w", err)
	}

	log.Printf("control connected server=%s", client.config.ServerAddr)

	reader := bufio.NewReader(conn)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("read control message: %w", readErr)
		}

		kind, connID, parseErr := protocol.ParseControlMessage(line)
		if parseErr != nil {
			return parseErr
		}

		if kind == protocol.NewConnPrefix {
			go client.handleNewConn(connID)
		}
	}
}

func (client *Client) handleNewConn(connID string) {
	dataConn, err := net.Dial("tcp", client.config.ServerAddr)
	if err != nil {
		log.Printf("dial data channel failed connID=%s error=%v", connID, err)
		return
	}

	if _, err := dataConn.Write([]byte(protocol.BuildDataHello(connID))); err != nil {
		log.Printf("send data hello failed connID=%s error=%v", connID, err)
		_ = dataConn.Close()
		return
	}

	targetConn, err := net.Dial("tcp", client.config.TargetAddr)
	if err != nil {
		log.Printf("dial target failed connID=%s target=%s error=%v", connID, client.config.TargetAddr, err)
		_ = dataConn.Close()
		return
	}

	log.Printf("relay start connID=%s target=%s", connID, client.config.TargetAddr)
	transport.Relay(dataConn, targetConn)
	log.Printf("relay end connID=%s", connID)
}
