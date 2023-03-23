package server

import (
	"io"
)

type Encoder interface {
	Encode(*TCPSession, []byte) ([]byte, error)
}

type Decoder interface {
	Decode(*TCPSession, io.Reader) ([]byte, error)
}
