package server

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/j178/reverse-tunnel/internal/protocol"
	"github.com/j178/reverse-tunnel/internal/transport"
)

type Config struct {
	ListenPublic string
	ListenClient string
	PairTimeout  time.Duration
}

type Server struct {
	config Config

	nextID atomic.Uint64

	mu          sync.Mutex
	controlConn net.Conn
	controlRW   *bufio.ReadWriter
	pending     map[string]chan net.Conn
}

func New(config Config) *Server {
	return &Server{
		config:  config,
		pending: make(map[string]chan net.Conn),
	}
}

func (server *Server) Run() error {
	publicListener, err := net.Listen("tcp", server.config.ListenPublic)
	if err != nil {
		return fmt.Errorf("listen public: %w", err)
	}
	defer publicListener.Close()

	clientListener, err := net.Listen("tcp", server.config.ListenClient)
	if err != nil {
		return fmt.Errorf("listen client: %w", err)
	}
	defer clientListener.Close()

	log.Printf("server listening public=%s client=%s", server.config.ListenPublic, server.config.ListenClient)

	go server.acceptClientSide(clientListener)

	for {
		conn, acceptErr := publicListener.Accept()
		if acceptErr != nil {
			return fmt.Errorf("accept public: %w", acceptErr)
		}

		go server.handlePublicConn(conn)
	}
}

func (server *Server) acceptClientSide(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept client side failed: %v", err)
			return
		}

		go server.handleClientConn(conn)
	}
}

func (server *Server) handleClientConn(conn net.Conn) {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("set read deadline failed: %v", err)
		_ = conn.Close()
		return
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("read hello failed: %v", err)
		_ = conn.Close()
		return
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("clear read deadline failed: %v", err)
		_ = conn.Close()
		return
	}

	kind, connID, err := protocol.ParseHello(line)
	if err != nil {
		log.Printf("invalid hello: %v", err)
		_ = conn.Close()
		return
	}

	switch kind {
	case protocol.ControlHelloPrefix:
		server.registerControl(conn, reader)
	case protocol.DataHelloPrefix:
		server.matchDataConn(connID, conn)
	default:
		_ = conn.Close()
	}
}

func (server *Server) registerControl(conn net.Conn, existingReader *bufio.Reader) {
	server.mu.Lock()
	if server.controlConn != nil {
		server.mu.Unlock()
		log.Printf("reject extra control client from %s", conn.RemoteAddr())
		_ = conn.Close()
		return
	}

	server.controlConn = conn
	server.controlRW = bufio.NewReadWriter(existingReader, bufio.NewWriter(conn))
	server.mu.Unlock()

	log.Printf("control client online: %s", conn.RemoteAddr())

	buffer := make([]byte, 1)
	_, _ = conn.Read(buffer)

	server.mu.Lock()
	if server.controlConn == conn {
		server.controlConn = nil
		server.controlRW = nil
	}
	server.mu.Unlock()

	_ = conn.Close()
	log.Printf("control client offline: %s", conn.RemoteAddr())
}

func (server *Server) handlePublicConn(publicConn net.Conn) {
	connID := server.newConnID()

	dataConnCh := make(chan net.Conn, 1)
	server.mu.Lock()
	server.pending[connID] = dataConnCh
	server.mu.Unlock()

	if err := server.sendNewConn(connID); err != nil {
		server.deletePending(connID)
		log.Printf("notify client failed connID=%s error=%v", connID, err)
		_ = publicConn.Close()
		return
	}

	log.Printf("public accepted connID=%s remote=%s", connID, publicConn.RemoteAddr())

	var dataConn net.Conn
	select {
	case dataConn = <-dataConnCh:
	case <-time.After(server.config.PairTimeout):
		server.deletePending(connID)
		log.Printf("pair timeout connID=%s", connID)
		_ = publicConn.Close()
		return
	}

	log.Printf("pair success connID=%s", connID)
	transport.Relay(publicConn, dataConn)
	log.Printf("relay closed connID=%s", connID)
}

func (server *Server) sendNewConn(connID string) error {
	server.mu.Lock()
	defer server.mu.Unlock()

	if server.controlRW == nil {
		return errors.New("no active control client")
	}

	if _, err := server.controlRW.WriteString(protocol.BuildNewConn(connID)); err != nil {
		return err
	}

	if err := server.controlRW.Flush(); err != nil {
		return err
	}

	return nil
}

func (server *Server) matchDataConn(connID string, dataConn net.Conn) {
	server.mu.Lock()
	dataConnCh, found := server.pending[connID]
	if found {
		delete(server.pending, connID)
	}
	server.mu.Unlock()

	if !found {
		log.Printf("unknown connID on data channel: %s", connID)
		_ = dataConn.Close()
		return
	}

	select {
	case dataConnCh <- dataConn:
	default:
		_ = dataConn.Close()
	}
}

func (server *Server) deletePending(connID string) {
	server.mu.Lock()
	delete(server.pending, connID)
	server.mu.Unlock()
}

func (server *Server) newConnID() string {
	sequence := server.nextID.Add(1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), sequence)
}
