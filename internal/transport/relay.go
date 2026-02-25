package transport

import (
	"io"
	"net"
	"sync"
)

func Relay(left net.Conn, right net.Conn) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(left, right)
		closeWrite(left)
	}()

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(right, left)
		closeWrite(right)
	}()

	waitGroup.Wait()
	_ = left.Close()
	_ = right.Close()
}

func closeWrite(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.CloseWrite()
		return
	}
	_ = conn.Close()
}
