package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"example.com/mq/logger"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ClientManager represents a connected client with its AckManager.
type ClientManager struct {
	conn       *websocket.Conn
	ackManager *MessageAckManager
	topics     map[string]bool
}

// Server represents the WebSocket server.
type Server struct {
	mu      sync.Mutex
	clients map[*ClientManager]bool
	topics  map[string]*Topic
	logger  *zap.Logger
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
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	client := &ClientManager{
		conn:       conn,
		ackManager: NewMessageAckManager(),
		topics:     make(map[string]bool),
	}

	// Add client to the server's client list.
	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
		client.conn.Close()
	}()

	s.logger.Info("Client connected", zap.String("client", conn.RemoteAddr().String()))

	// Listen for client messages.
	s.listenToClient(client)
}

// listenToClient listens for incoming messages from a client.
func (s *Server) listenToClient(client *ClientManager) {
	for {
		var msg struct {
			Action string `json:"action"`
			Topic  string `json:"topic"`
			Data   string `json:"data"`
		}

		err := client.conn.ReadJSON(&msg)
		if err != nil {
			s.logger.Error("Failed to read message", zap.Error(err))
			return
		}

		switch msg.Action {
		case "subscribe":
			s.subscribeClientToTopic(client, msg.Topic)
		case "publish":
			s.publishToTopic(client, msg.Topic, msg.Data)
		default:
			s.logger.Warn("Unknown action", zap.String("action", msg.Action))
		}
	}
}

// subscribeClientToTopic subscribes a client to a specific topic.
func (s *Server) subscribeClientToTopic(client *ClientManager, topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.topics[topic]; !exists {
		s.logger.Warn("Topic not found", zap.String("topic", topic))
		return
	}

	client.topics[topic] = true
	s.logger.Info("Client subscribed to topic", zap.String("client", client.conn.RemoteAddr().String()), zap.String("topic", topic))
}

// publishToTopic handles publishing a message to a topic.
func (s *Server) publishToTopic(client *ClientManager, topic, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.topics[topic]; !exists {
		s.logger.Warn("Topic not found", zap.String("topic", topic))
		return
	}

	t := s.topics[topic]
	t.Publish(data)

	// Send ack back to client.
	ackMsg := fmt.Sprintf("Ack for message on topic %s", topic)
	err := client.conn.WriteMessage(websocket.TextMessage, []byte(ackMsg))
	if err != nil {
		s.logger.Error("Failed to send ack", zap.Error(err))
	}
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

	logger.Info("Server starting", zap.String("addr", ":8080"))
	server.Run(":8080")
}
