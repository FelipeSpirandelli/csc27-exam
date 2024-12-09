package main

import (
	"sync"

	"example.com/mq/logger"
	"go.uber.org/zap"
)

// Topic represents a message topic with a queue and subscribers.
type Topic struct {
	name        string
	subscribers map[*ClientManager]bool
	messages    []string
	mu          sync.Mutex
}

// NewTopic creates a new Topic instance.
func NewTopic(name string) *Topic {
	return &Topic{
		name:        name,
		subscribers: make(map[*ClientManager]bool),
		messages:    []string{},
	}
}

// Publish adds a message to the topic and sends it to all subscribers.
func (t *Topic) Publish(message string) {
	t.mu.Lock()
	t.messages = append(t.messages, message)
	t.mu.Unlock()
}

// AddSubscriber adds a client to the topic's subscriber list.
func (t *Topic) AddSubscriber(client *ClientManager) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subscribers[client] = true
	logger.Info("Subscriber added", zap.String("topic", t.name), zap.String("client", client.conn.RemoteAddr().String()))
}

// RemoveSubscriber removes a client from the topic's subscriber list.
func (t *Topic) RemoveSubscriber(client *ClientManager) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.subscribers, client)
	logger.Info("Subscriber removed", zap.String("topic", t.name), zap.String("client", client.conn.RemoteAddr().String()))
}
