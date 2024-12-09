package main

import (
	"sync"
	"time"
)

// MessageAckManager manages acks for incoming and outgoing messages.
type MessageAckManager struct {
	mu                  sync.Mutex
	pendingInboundAcks  map[uint64]time.Time // Tracks inbound messages awaiting processing ack
	pendingOutboundAcks map[uint64]time.Time // Tracks outbound messages awaiting client ack
}

// NewMessageAckManager creates a new MessageAckManager.
//
// Returns:
//
//	*MessageAckManager: A newly initialized ack manager with empty maps.
func NewMessageAckManager() *MessageAckManager {
	return &MessageAckManager{
		pendingInboundAcks:  make(map[uint64]time.Time),
		pendingOutboundAcks: make(map[uint64]time.Time),
	}
}

// AddInboundAck adds an inbound message (from client to server) to tracking.
//
// Args:
//
//	msgID: A unique identifier for the message.
func (m *MessageAckManager) AddInboundAck(msgID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingInboundAcks[msgID] = time.Now()
}

// MarkInboundAcked removes an inbound message that has been processed.
//
// Args:
//
//	msgID: The unique identifier of the message to mark acked.
func (m *MessageAckManager) MarkInboundAcked(msgID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pendingInboundAcks, msgID)
}

// AddOutboundAck adds an outbound message (from server to client) to tracking.
//
// Args:
//
//	msgID: A unique identifier for the message.
func (m *MessageAckManager) AddOutboundAck(msgID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingOutboundAcks[msgID] = time.Now()
}

// MarkOutboundAcked removes an outbound message that has been acknowledged by the client.
//
// Args:
//
//	msgID: The unique identifier of the message to mark acked.
func (m *MessageAckManager) MarkOutboundAcked(msgID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pendingOutboundAcks, msgID)
}

// GetUnackedOutbound returns the IDs of outbound messages that haven't been acked yet.
//
// Returns:
//
//	[]uint64: A slice of message IDs that are still waiting for an ack.
func (m *MessageAckManager) GetUnackedOutbound() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]uint64, 0, len(m.pendingOutboundAcks))
	for id := range m.pendingOutboundAcks {
		ids = append(ids, id)
	}
	return ids
}
