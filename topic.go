package main

import (
	"sync"

	"example.com/mq/logger"
	"go.uber.org/zap"
)

type Topic struct {
	name        string
	subscribers map[*ClientManager]bool
	messages    []string
	mu          sync.Mutex
}

func NewTopic(name string) *Topic {
	return &Topic{
		name:        name,
		subscribers: make(map[*ClientManager]bool),
		messages:    []string{},
	}
}

func (t *Topic) Publish(message string, transactionId string) {
	t.mu.Lock()
	t.messages = append(t.messages, message)
	t.mu.Unlock()

	logger.Debug("Message queued in topic",
		zap.String("transactionId", transactionId),
		zap.String("status", "data"),
		zap.String("topic", t.name))
}

func (t *Topic) AddSubscriber(client *ClientManager, transactionId string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subscribers[client] = true
	logger.Info("Subscriber added",
		zap.String("transactionId", transactionId),
		zap.String("status", "commit"),
		zap.String("topic", t.name),
		zap.String("client", client.conn.RemoteAddr().String()))
}

func (t *Topic) RemoveSubscriber(client *ClientManager, transactionId string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.subscribers, client)
	logger.Info("Subscriber removed",
		zap.String("transactionId", transactionId),
		zap.String("status", "commit"),
		zap.String("topic", t.name),
		zap.String("client", client.conn.RemoteAddr().String()))
}
