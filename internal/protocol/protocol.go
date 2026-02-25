package protocol

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ControlHelloPrefix = "CONTROL"
	DataHelloPrefix    = "DATA"
	NewConnPrefix      = "NEW"
)

func BuildControlHello() string {
	return ControlHelloPrefix + "\n"
}

func BuildDataHello(connID string) string {
	return fmt.Sprintf("%s %s\n", DataHelloPrefix, connID)
}

func BuildNewConn(connID string) string {
	return fmt.Sprintf("%s %s\n", NewConnPrefix, connID)
}

func ParseHello(line string) (kind string, connID string, err error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", "", errors.New("empty hello")
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 1 && parts[0] == ControlHelloPrefix {
		return ControlHelloPrefix, "", nil
	}

	if len(parts) == 2 && parts[0] == DataHelloPrefix {
		if parts[1] == "" {
			return "", "", errors.New("missing connID")
		}
		return DataHelloPrefix, parts[1], nil
	}

	return "", "", fmt.Errorf("invalid hello: %s", trimmed)
}

func ParseControlMessage(line string) (kind string, connID string, err error) {
	trimmed := strings.TrimSpace(line)
	parts := strings.Fields(trimmed)
	if len(parts) == 2 && parts[0] == NewConnPrefix {
		return NewConnPrefix, parts[1], nil
	}

	return "", "", fmt.Errorf("invalid control message: %s", trimmed)
}
