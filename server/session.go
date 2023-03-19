package server

import (
	"github.com/bruuuces/gnet-tcp/message"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"time"
)

type TCPSession struct {
	id      uint64
	conn    net.Conn
	server  *TCPServer
	decoder message.Decoder
	encoder message.Encoder
	handler Handler
	close   chan bool
	sndBuf  chan []byte
	// 读超时时间
	ReadTimeOut time.Duration
	// 业务消息发送缓冲区大小
	SendBufferSize int
	// 自定义属性
	Attributes map[string]any
}

func NewTCPSession(conn net.Conn, server *TCPServer) *TCPSession {
	session := TCPSession{
		conn:       conn,
		server:     server,
		decoder:    server.Decoder(),
		encoder:    server.Encoder(),
		handler:    server.Handler(),
		close:      make(chan bool, 1),
		Attributes: make(map[string]any),
	}
	session.SendBufferSize = -1
	return &session
}

func (s *TCPSession) Open() {
	sendBufferSize := s.SendBufferSize
	if sendBufferSize <= 0 {
		sendBufferSize = 1024
	}
	s.SendBufferSize = sendBufferSize
	s.sndBuf = make(chan []byte, s.SendBufferSize)
	s.server.OnSessionOpen(s)
	go s.HandleRead()
	go s.HandleWrite()
}

func (s *TCPSession) HandleRead() {
	defer func(s *TCPSession) {
		_ = s.Close()
	}(s)
	for {
		readTimeOut := s.ReadTimeOut
		if readTimeOut > 0 {
			err := s.conn.SetReadDeadline(time.Now().Add(readTimeOut))
			if err != nil {
				log.Error().
					Err(err).
					Uint64("ssid", s.GetID()).
					Msgf("set read deadline error")
				return
			}
		}
		packet, err := s.decoder.Decode(s, s.conn)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				log.Error().
					Err(err).
					Uint64("ssid", s.GetID()).
					Msg("read message error")

			}
			return
		}
		log.Trace().
			Uint64("ssid", s.GetID()).
			Hex("packet", packet).
			Msg("decode message")
		s.handler.Process(s, packet)
	}
}

func (s *TCPSession) HandleWrite() {
	defer func(s *TCPSession) {
		_ = s.Close()
	}(s)
	for {
		select {
		case packet := <-s.sndBuf:
			log.Trace().
				Uint64("ssid", s.GetID()).
				Hex("packet", packet).
				Msg("encode message")
			msgData, err := s.encoder.Encode(s, packet)
			var writeLen int
			if err == nil {
				writeLen, err = s.conn.Write(msgData)
			}
			if err == io.EOF {
				log.Error().
					Err(err).
					Uint64("ssid", s.GetID()).
					Msg("write message error, connection closed")
				return
			} else if err != nil {
				log.Error().
					Err(err).
					Uint64("ssid", s.GetID()).
					Int("writeLen", writeLen).
					Msg("write message error")
				return
			} else if writeLen != len(msgData) {
				log.Error().
					Uint64("ssid", s.GetID()).
					Int("writeLen", writeLen).
					Int("msgDataLen", len(msgData)).
					Msg("write message error, length field length error")
				return
			}
		case <-s.close:
			return
		}
	}
}

func (s *TCPSession) Send(picket []byte) {
	s.sndBuf <- picket
}

func (s *TCPSession) Close() error {
	s.close <- true
	conn := s.conn
	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (s *TCPSession) GetID() uint64 {
	return s.id
}

func (s *TCPSession) GetConn() any {
	return s.conn
}

func (s *TCPSession) GetServer() *TCPServer {
	return s.server
}

func (s *TCPSession) GetRemoteAddr() string {
	conn := s.conn
	if conn == nil {
		return ""
	}
	return conn.RemoteAddr().String()
}

func (s *TCPSession) GetLocalAddr() string {
	conn := s.conn
	if conn == nil {
		return ""
	}
	return conn.LocalAddr().String()
}
