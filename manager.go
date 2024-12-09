package main

import (
	"sync"
)

type MessageAckManager struct {
	mu                 sync.Mutex
	lastQueueAckedMap  map[string]int
	lastClientAckedMap map[string]int
}

func NewMessageAckManager() *MessageAckManager {
	return &MessageAckManager{
		lastQueueAckedMap:  make(map[string]int),
		lastClientAckedMap: make(map[string]int),
	}
}

func (m *MessageAckManager) GetLastQueueAcked(topic string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastQueueAckedMap[topic]
}

func (m *MessageAckManager) SetLastQueueAcked(topic string, no int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastQueueAckedMap[topic] = no
}

func (m *MessageAckManager) GetLastClientAcked(topic string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastClientAckedMap[topic]
}

func (m *MessageAckManager) SetLastClientAcked(topic string, no int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastClientAckedMap[topic] = no
}
