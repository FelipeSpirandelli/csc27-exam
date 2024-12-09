package main

import (
	"sync"
)

type MessageAckManager struct {
	mu           sync.Mutex
	lastAckedMap map[string]int
}

func NewMessageAckManager() *MessageAckManager {
	return &MessageAckManager{
		lastAckedMap: make(map[string]int),
	}
}

func (m *MessageAckManager) GetLastAcked(topic string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastAckedMap[topic]
}

func (m *MessageAckManager) SetLastAcked(topic string, no int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastAckedMap[topic] = no
}
