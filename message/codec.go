package message

import (
	"github.com/bruuuces/gnet-tcp/server"
	"io"
)

type Encoder interface {
	Encode(*server.TCPSession, []byte) ([]byte, error)
}

type Decoder interface {
	Decode(*server.TCPSession, io.Reader) ([]byte, error)
}
