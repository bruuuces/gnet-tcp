package server

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"runtime/debug"
	"sync"
	"time"
)

type TCPServerConfigurator struct {
	NoDelay            bool
	KeepAlive          bool
	KeepAlivePeriodSec int
	MaxConnectionNum   int
	ReadTimeOutSec     int
	SendBufferSize     int
}

func DefaultTCPServerConfigurator() TCPServerConfigurator {
	return TCPServerConfigurator{
		NoDelay:            true,
		KeepAlive:          true,
		KeepAlivePeriodSec: int(time.Minute.Seconds()),
		MaxConnectionNum:   10000,
		ReadTimeOutSec:     10,
		SendBufferSize:     1024,
	}
}

type TCPServer struct {
	addr              string
	config            TCPServerConfigurator
	ln                *net.TCPListener
	mutexSession      sync.Mutex
	lnClosedWait      sync.WaitGroup
	sessionClosedWait sync.WaitGroup
	SessionMgr        *SessionManager
	Encoder           func() Encoder
	Decoder           func() Decoder
	Handler           func() Handler
	OnSessionOpen     func(*TCPSession)
	OnSessionClose    func(*TCPSession)
}

func NewTCPServer(addr string, config TCPServerConfigurator) *TCPServer {
	return &TCPServer{
		addr:       addr,
		config:     config,
		SessionMgr: &SessionManager{},
	}
}

func (s *TCPServer) Init() {
	addr, err := net.ResolveTCPAddr("tcp4", s.addr)
	if err != nil {
		panic(fmt.Errorf("addr resolve error, addr:%v, err:%v", addr, err))
	}
	ln, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		panic(fmt.Errorf("listen error, err:%v", err))
	}
	s.ln = ln
	s.SessionMgr.Init()
	log.Info().
		Str("addr", s.addr).
		Msgf("tcp server listened")
}

func (s *TCPServer) Start() {
	defer func() {
		if v := recover(); v != nil {
			log.Error().Msgf("tcp server panic %v %v", v, string(debug.Stack()))
		}
	}()
	s.lnClosedWait.Add(1)
	defer s.lnClosedWait.Done()
	for {
		conn, err := s.ln.AcceptTCP()
		if err != nil {
			log.Error().
				Err(err).
				Msg("accept error")
			break
		}
		err = conn.SetKeepAlive(s.config.KeepAlive)
		if err != nil {
			log.Error().
				Err(err).
				Msg("SetKeepAlive error")
			continue
		}
		if s.config.KeepAlive {
			err = conn.SetKeepAlivePeriod(time.Duration(s.config.KeepAlivePeriodSec) * time.Second)
			if err != nil {
				log.Error().
					Err(err).
					Msg("SetKeepAlivePeriod error")
				continue
			}
		}
		err = conn.SetNoDelay(s.config.NoDelay)
		if err != nil {
			log.Error().
				Err(err).
				Msg("SetNoDelay error")
			continue
		}
		session := s.bindSession(conn)
		go session.Open()
	}
}

func (s *TCPServer) bindSession(conn *net.TCPConn) *TCPSession {
	session := NewTCPSession(conn, s)
	session.ReadTimeOut = time.Duration(s.config.ReadTimeOutSec) * time.Second
	s.mutexSession.Lock()
	defer s.mutexSession.Unlock()
	s.SessionMgr.registerSession(session)
	return session
}

func (s *TCPServer) Stop() {
	err := s.ln.Close()
	if err != nil {
		log.Error().
			Err(err).
			Msg("close listener error")
	}
	s.lnClosedWait.Wait()
	s.mutexSession.Lock()
	defer s.mutexSession.Unlock()
	s.SessionMgr.Close()
	s.sessionClosedWait.Wait()
}

func (s *TCPServer) removeSession(session *TCPSession) {
	err := session.Close()
	if err != nil {
		log.Error().
			Err(err).
			Uint64("ssid", session.GetID()).
			Msg("close session error")
	}
	s.mutexSession.Lock()
	defer s.mutexSession.Unlock()
	s.SessionMgr.unregisterSession(session)
}

type Handler interface {
	Process(*TCPSession, []byte)
}
