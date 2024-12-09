package main

import (
	"sync"
)

// MessageAckManager manages last acked messages per topic for a client.
type MessageAckManager struct {
	mu           sync.Mutex
	lastAckedMap map[string]int
}

// NewMessageAckManager initializes a new MessageAckManager.
func NewMessageAckManager() *MessageAckManager {
	return &MessageAckManager{
		lastAckedMap: make(map[string]int),
	}
}

// GetLastAcked returns the last acked message number for a given topic.
func (m *MessageAckManager) GetLastAcked(topic string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastAckedMap[topic]
}

// SetLastAcked sets the last acked message number for a given topic.
func (m *MessageAckManager) SetLastAcked(topic string, no int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastAckedMap[topic] = no
}
