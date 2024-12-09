package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"example.com/mq/logger"
	"example.com/mq/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ClientManager represents a connected client.
type ClientManager struct {
	conn                  *websocket.Conn
	ackManager            *MessageAckManager
	topics                map[string]bool
	mu                    sync.Mutex
	sendingRoutineStarted bool
}

// Server represents the WebSocket server.
type Server struct {
	mu      sync.Mutex
	clients map[*ClientManager]bool
	topics  map[string]*Topic
}

// NewServer initializes a new Server.
func NewServer() *Server {
	return &Server{
		clients: make(map[*ClientManager]bool),
		topics:  make(map[string]*Topic),
	}
}

// HandleClient manages an individual client connection.
func (s *Server) HandleClient(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	client := &ClientManager{
		conn:       conn,
		ackManager: NewMessageAckManager(),
		topics:     make(map[string]bool),
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
		client.conn.Close()
	}()

	logger.Info("Client connected", zap.String("client", conn.RemoteAddr().String()))
	s.listenToClient(client)
}

// BaseMessage basic struct for action/topic messages.
type BaseMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
}

// PublishMessage includes Data and ClientMessageNo.
type PublishMessage struct {
	BaseMessage
	Data            string `json:"data"`
	ClientMessageNo int    `json:"clientMessageNo"`
}

type AckPublishMessage struct {
	BaseMessage
	ClientMessageNo int `json:"clientMessageNo"`
}

// SubscribeMessage includes only Action and Topic.
type SubscribeMessage struct {
	BaseMessage
}

// AckMessage includes QueueMessageNo.
type AckMessage struct {
	BaseMessage
	QueueMessageNo int `json:"queueMessageNo"`
}

// listenToClient listens for incoming messages from a client.
func (s *Server) listenToClient(client *ClientManager) {
	for {
		_, msgBytes, err := client.conn.ReadMessage()
		if err != nil {
			logger.Error("Failed to read message", zap.Error(err))
			return
		}

		var base BaseMessage
		if err := json.Unmarshal(msgBytes, &base); err != nil {
			logger.Warn("Failed to parse base message", zap.Error(err))
			continue
		}

		switch base.Action {
		case "subscribe":
			var msg SubscribeMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				logger.Warn("Failed to parse subscribe message", zap.Error(err))
				continue
			}
			s.subscribeClientToTopic(client, msg.Topic)

		case "publish":
			var msg PublishMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				logger.Warn("Failed to parse publish message", zap.Error(err))
				continue
			}
			if client.ackManager.GetLastAcked(msg.Topic)+1 != msg.ClientMessageNo {
				logger.Warn("Ignoring out-of-order message", zap.String("topic", msg.Topic), zap.Int("clientMessageNo", msg.ClientMessageNo))
				continue
			}
			s.publishToTopic(msg.Topic, msg.Data)
			s.ackPublishToTopic(client, msg.Topic, msg.ClientMessageNo)
		case "ack":
			var msg AckMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				logger.Warn("Failed to parse ack message", zap.Error(err))
				continue
			}
			s.treatAckMessage(client, msg.Topic, msg.QueueMessageNo)
		default:
			logger.Warn("Unknown action", zap.String("action", base.Action))
		}
	}
}

// subscribeClientToTopic subscribes a client to a specific topic.
// It also ensures that the unacked messages routine is started only once.
func (s *Server) subscribeClientToTopic(client *ClientManager, topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.topics[topic]
	if !exists {
		logger.Warn("Topic not found", zap.String("topic", topic))
		return
	}

	client.topics[topic] = true
	t.AddSubscriber(client)
	logger.Info("Client subscribed to topic", zap.String("client", client.conn.RemoteAddr().String()), zap.String("topic", topic))

	// Start sending routine for client if not already started
	if !client.sendingRoutineStarted {
		client.sendingRoutineStarted = true
		go s.sendUnackedMessagesRoutine(client)
	}
}

// publishToTopic handles publishing a message to a topic.
// IMPORTANT: Now we only enqueue the message.
func (s *Server) publishToTopic(topic, data string) {
	s.mu.Lock()
	t, exists := s.topics[topic]
	s.mu.Unlock()

	if !exists {
		logger.Warn("Topic not found", zap.String("topic", topic))
		return
	}

	// Add the message to the topic queue (no immediate sending)
	t.Publish(data)
}

func (s *Server) ackPublishToTopic(client *ClientManager, topic string, clientMessageNo int) {
	// Send ack back to publisher to confirm reception of their message
	ack := AckPublishMessage{
		BaseMessage:     BaseMessage{Action: "ack", Topic: topic},
		ClientMessageNo: clientMessageNo,
	}
	ackBytes, err := json.Marshal(ack)
	if err != nil {
		logger.Errorf("Failed to marshal ack message: %v", err)
		return
	}
	go utils.ExponentialBackoff(func() error {
		return client.conn.WriteMessage(websocket.TextMessage, ackBytes)
	}, 5, 400*time.Millisecond, 20*time.Second)
}

// treatAckMessage updates the last acked message number for that client and topic.
func (s *Server) treatAckMessage(client *ClientManager, topic string, queueNo int) {
	client.ackManager.SetLastAcked(topic, queueNo)
	logger.Info("Message acked", zap.String("topic", topic), zap.Int("queueNo", queueNo))
}

func (s *Server) sendUnackedMessagesRoutine(client *ClientManager) {
	baseDelay := 400 * time.Millisecond
	maxDelay := 40 * time.Second
	delay := baseDelay

	for {
		client.mu.Lock()
		// Collect unacked messages from all subscribed topics
		var msgsToSend []queuedMessage
		for topicName := range client.topics {
			s.mu.Lock()
			t, exists := s.topics[topicName]
			s.mu.Unlock()
			if !exists {
				continue
			}

			t.mu.Lock()
			lastAcked := client.ackManager.GetLastAcked(topicName)
			for i := lastAcked; i < len(t.messages); i++ {
				msgsToSend = append(msgsToSend, queuedMessage{
					Action: "send",
					Topic:  topicName,
					No:     i + 1,
					Data:   t.messages[i],
				})
			}
			t.mu.Unlock()
		}
		client.mu.Unlock()

		// If no messages, reset delay and sleep
		if len(msgsToSend) == 0 {
			delay = baseDelay
			time.Sleep(delay)
			continue
		}

		// Attempt to send all unacked messages as one batch
		err := sendAllMessagesOnce(client, msgsToSend)
		if err == nil {
			// Success: reset delay and wait before next check
			delay = baseDelay
			time.Sleep(delay)
			continue
		}

		// Failed to send: apply exponential backoff
		delay = delay * 2
		if delay > maxDelay {
			delay = maxDelay
		}
		time.Sleep(delay)
	}
}

// sendAllMessagesOnce attempts to send a batch of messages once.
func sendAllMessagesOnce(client *ClientManager, msgs []queuedMessage) error {
	for _, qm := range msgs {
		qmMsgBytes, err := json.Marshal(qm)
		if err != nil {
			logger.Errorf("Failed to marshal qm message: %v", err)
			return err
		}
		if err := client.conn.WriteMessage(websocket.TextMessage, qmMsgBytes); err != nil {
			return err
		}
	}
	return nil
}

type queuedMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
	No     int    `json:"queueMessageNo"`
	Data   string `json:"data"`
}

// Run starts the WebSocket server.
func (s *Server) Run(addr string) {
	http.HandleFunc("/ws", s.HandleClient)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func main() {
	defer logger.Sync()

	logger.Info("Application started",
		zap.String("version", "1.0.0"),
		zap.Time("timestamp", time.Now()),
	)

	server := NewServer()

	// Initialize some topics.
	server.topics["news"] = NewTopic("news")
	server.topics["sports"] = NewTopic("sports")
	server.topics["tech"] = NewTopic("tech")

	logger.Info("Server starting", zap.String("addr", ":8082"))
	server.Run(":8082")
}
