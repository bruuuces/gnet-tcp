package server

import (
	"github.com/rs/zerolog/log"
	"sync/atomic"
)

type SessionManager struct {
	ssidSeed       uint64
	sessionCounter int64
	sessionMap     map[uint64]*TCPSession
}

func (m *SessionManager) Init() {
	m.ssidSeed = 0
	m.sessionCounter = 0
	m.sessionMap = make(map[uint64]*TCPSession)
}

func (m *SessionManager) Close() {
	for _, session := range m.sessionMap {
		err := session.Close()
		if err != nil {
			log.Error().
				Err(err).
				Uint64("ssid", session.GetID()).
				Msg("close session error")
		}
	}
	m.sessionMap = nil
}

func (m *SessionManager) registerSession(session *TCPSession) {
	ssid := atomic.AddUint64(&m.ssidSeed, 1)
	session.id = ssid
	m.sessionMap[session.GetID()] = session
}

func (m *SessionManager) unregisterSession(session *TCPSession) {
	atomic.AddInt64(&m.sessionCounter, -1)
	delete(m.sessionMap, session.GetID())
}

func (m *SessionManager) GetSession(ssid uint64) (*TCPSession, bool) {
	session, ok := m.sessionMap[ssid]
	return session, ok
}

func (m *SessionManager) SessionCount() int64 {
	return atomic.LoadInt64(&m.sessionCounter)
}

func (m *SessionManager) SessionMap() map[uint64]*TCPSession {
	return m.sessionMap
}
