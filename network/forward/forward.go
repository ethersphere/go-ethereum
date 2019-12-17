package forward

import (
	"sync"

	"github.com/ethersphere/swarm/network"
)

type Session struct {
	kademlia        *network.Kademlia
	pivot           []byte
	id              int
	capabilityIndex string
}

type SessionManager struct {
	sessions []*Session
	mu       sync.Mutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

func (m *SessionManager) add(s *Session) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.id = len(m.sessions)
	m.sessions = append(m.sessions, s)
	return s
}

func (m *SessionManager) New(kad *network.Kademlia, capabilityIndex string, pivot []byte) *Session {
	s := &Session{
		kademlia:        kad,
		capabilityIndex: capabilityIndex,
	}
	if pivot == nil {
		s.pivot = kad.BaseAddr()
	} else {
		s.pivot = pivot
	}
	return m.add(s)
}

//func NewFromContext(sctx *SessionContext, kad *network.Kademlia) *Session {
//	s := &Session{
//		kademlia: kad,
//	}
//
//	s.id = sctx.Value("id").(int)
//
//	addr := sctx.Value("address")
//	if addr == nil {
//		s.pivot = kad.BaseAddr()
//	} else {
//		s.pivot = addr.([]byte)
//	}
//
//	capabilityIndex := sctx.Value("capability")
//	if capabilityIndex != nil {
//		s.capabilityIndex = capabilityIndex.(string)
//	}
//
//	return s
//}

func (s *Session) Get(numPeers int) ([]ForwardPeer, error) {
	var result []ForwardPeer

	return result, nil
}
