package main

import (
	"github.com/bruuuces/gnet-tcp/server"
	"time"
)

func main() {
	config := server.TCPServerConfigurator{
		NoDelay:            true,
		KeepAlive:          true,
		KeepAlivePeriodSec: int(time.Minute.Seconds()),
		SendBufferSize:     1,
		ReadTimeOutSec:     10,
	}
	tcpServer := server.NewTCPServer(":10001", config)
	tcpServer.Encoder = func() server.Encoder {
		encoder, err := server.NewLengthFieldPrepender(4)
		if err != nil {
			panic(err)
		}
		return encoder
	}
	tcpServer.Decoder = func() server.Decoder {
		decoder, err := server.NewLengthFieldBasedFrameDecoder(4, 4*1024*1024)
		if err != nil {
			panic(err)
		}
		return decoder
	}
	tcpServer.Handler = func() server.Handler {
		return &TestMessageHandler{}
	}
	tcpServer.Init()
	tcpServer.Start()
}

type TestMessageHandler struct {
}

func (h *TestMessageHandler) Process(session *server.TCPSession, packet []byte) {
	session.Send(packet)
}
