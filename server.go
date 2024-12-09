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

type ClientManager struct {
	conn                  *websocket.Conn
	ackManager            *MessageAckManager
	topics                map[string]bool
	mu                    sync.Mutex
	sendingRoutineStarted bool
}

type Server struct {
	mu      sync.Mutex
	clients map[*ClientManager]bool
	topics  map[string]*Topic
}

type BaseMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
}

type PublishMessage struct {
	BaseMessage
	Data            string `json:"data"`
	ClientMessageNo int    `json:"clientMessageNo"`
}

type AckPublishMessage struct {
	BaseMessage
	ClientMessageNo int `json:"clientMessageNo"`
}

type SubscribeMessage struct {
	BaseMessage
}

type AckMessage struct {
	BaseMessage
	QueueMessageNo int `json:"queueMessageNo"`
}

type queuedMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
	No     int    `json:"queueMessageNo"`
	Data   string `json:"data"`
}

func NewServer() *Server {
	return &Server{
		clients: make(map[*ClientManager]bool),
		topics:  make(map[string]*Topic),
	}
}

func (s *Server) HandleClient(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection",
			zap.Error(err))
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
		logger.Info("Client disconnected",
			zap.String("status", "finish"),
			zap.String("client", conn.RemoteAddr().String()))
	}()

	logger.Info("Client connected",
		zap.String("status", "begin"),
		zap.String("client", conn.RemoteAddr().String()))

	s.listenToClient(client)
	logger.Info("Finished client handling",
		zap.String("status", "finish"))
}

func (s *Server) listenToClient(client *ClientManager) {
	for {
		transactionId := utils.GenerateTransactionID()
		logger.Info("Waiting to receive new message",
			zap.String("transactionId", transactionId),
			zap.String("status", "begin"))

		_, msgBytes, err := client.conn.ReadMessage()
		if err != nil {
			// Determine if the error is due to a normal closure
			if closeErr, ok := err.(*websocket.CloseError); ok {
				switch closeErr.Code {
				case websocket.CloseNormalClosure, websocket.CloseGoingAway:
					logger.Info("Client closed the connection gracefully",
						zap.String("transactionId", transactionId),
						zap.Int("closeCode", closeErr.Code),
						zap.String("reason", closeErr.Text),
						zap.String("status", "info"))
				default:
					logger.Error("WebSocket closed with error",
						zap.String("transactionId", transactionId),
						zap.Int("closeCode", closeErr.Code),
						zap.String("reason", closeErr.Text),
						zap.String("status", "error"))
				}
			} else {
				// Handle other types of errors (e.g., network issues)
				logger.Error("Failed to read message",
					zap.String("transactionId", transactionId),
					zap.String("status", "error"),
					zap.Error(err))
			}
			return
		}

		var base BaseMessage
		if err := json.Unmarshal(msgBytes, &base); err != nil {
			logger.Warn("Failed to parse base message",
				zap.String("transactionId", transactionId),
				zap.String("status", "data"),
				zap.Error(err))
			continue
		}

		logger.Debug("Received message",
			zap.String("transactionId", transactionId),
			zap.String("status", "data"),
			zap.String("action", base.Action),
			zap.String("topic", base.Topic))

		switch base.Action {
		case "subscribe":
			var msg SubscribeMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				logger.Warn("Failed to parse subscribe message",
					zap.String("transactionId", transactionId),
					zap.String("status", "data"),
					zap.Error(err))
				continue
			}
			s.subscribeClientToTopic(client, msg.Topic, transactionId)

		case "publish":
			var msg PublishMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				logger.Warn("Failed to parse publish message",
					zap.String("transactionId", transactionId),
					zap.String("status", "data"),
					zap.Error(err))
				continue
			}
			if client.ackManager.GetLastClientAcked(msg.Topic)+1 != msg.ClientMessageNo {
				logger.Warn("Ignoring out-of-order message",
					zap.String("transactionId", transactionId),
					zap.String("status", "data"),
					zap.String("topic", msg.Topic),
					zap.Int("clientMessageNo", msg.ClientMessageNo))
				continue
			}
			s.publishToTopic(msg.Topic, msg.Data, transactionId)
			s.ackPublishToTopic(client, msg.Topic, msg.ClientMessageNo, transactionId)
			client.ackManager.SetLastClientAcked(msg.Topic, msg.ClientMessageNo)

		case "ack":
			var msg AckMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				logger.Warn("Failed to parse ack message",
					zap.String("transactionId", transactionId),
					zap.String("status", "data"),
					zap.Error(err))
				continue
			}
			s.treatAckMessage(client, msg.Topic, msg.QueueMessageNo, transactionId)

		default:
			logger.Warn("Unknown action",
				zap.String("transactionId", transactionId),
				zap.String("status", "data"),
				zap.String("action", base.Action))
		}

		logger.Info("Message processed",
			zap.String("transactionId", transactionId),
			zap.String("status", "commit"))
	}
}

func (s *Server) subscribeClientToTopic(client *ClientManager, topic string, transactionId string) {
	s.mu.Lock()
	t, exists := s.topics[topic]
	s.mu.Unlock()

	if !exists {
		logger.Warn("Topic not found",
			zap.String("transactionId", transactionId),
			zap.String("status", "data"),
			zap.String("topic", topic))
		return
	}

	client.topics[topic] = true
	t.AddSubscriber(client, transactionId)

	logger.Info("Client subscribed to topic",
		zap.String("transactionId", transactionId),
		zap.String("status", "commit"),
		zap.String("client", client.conn.RemoteAddr().String()),
		zap.String("topic", topic))

	if !client.sendingRoutineStarted {
		client.sendingRoutineStarted = true
		go s.sendUnackedMessagesRoutine(client)
	}
}

func (s *Server) publishToTopic(topic, data string, transactionId string) {
	s.mu.Lock()
	t, exists := s.topics[topic]
	s.mu.Unlock()

	if !exists {
		logger.Warn("Topic not found",
			zap.String("transactionId", transactionId),
			zap.String("status", "data"),
			zap.String("topic", topic))
		return
	}

	t.Publish(data, transactionId)
	logger.Info("Message published to topic",
		zap.String("transactionId", transactionId),
		zap.String("status", "commit"),
		zap.String("topic", topic))
}

func (s *Server) ackPublishToTopic(client *ClientManager, topic string, clientMessageNo int, transactionId string) {
	ack := AckPublishMessage{
		BaseMessage:     BaseMessage{Action: "ack", Topic: topic},
		ClientMessageNo: clientMessageNo,
	}
	ackBytes, err := json.Marshal(ack)
	if err != nil {
		logger.Error("Failed to marshal ack message",
			zap.String("transactionId", transactionId),
			zap.String("status", "error"),
			zap.Error(err))
		return
	}
	go utils.ExponentialBackoff(func() error {
		return client.conn.WriteMessage(websocket.TextMessage, ackBytes)
	}, 5, 400*time.Millisecond, 20*time.Second)
}

func (s *Server) treatAckMessage(client *ClientManager, topic string, queueNo int, transactionId string) {
	client.ackManager.SetLastQueueAcked(topic, queueNo)
	logger.Info("Message acked",
		zap.String("transactionId", transactionId),
		zap.String("status", "commit"),
		zap.String("topic", topic),
		zap.Int("queueNo", queueNo))
}

func (s *Server) sendUnackedMessagesRoutine(client *ClientManager) {
	baseDelay := 400 * time.Millisecond
	maxDelay := 40 * time.Second
	delay := baseDelay
	transactionId := utils.GenerateTransactionID()

	for {
		client.mu.Lock()
		var msgsToSend []queuedMessage
		for topicName := range client.topics {
			s.mu.Lock()
			t, exists := s.topics[topicName]
			s.mu.Unlock()
			if !exists {
				continue
			}

			t.mu.Lock()
			lastAcked := client.ackManager.GetLastQueueAcked(topicName)
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

		if len(msgsToSend) == 0 {
			time.Sleep(delay)
			continue
		}

		err := sendAllMessagesOnce(client, msgsToSend, transactionId)
		if err == nil {
			delay = baseDelay
			time.Sleep(delay)
			continue
		}

		delay = delay * 2
		if delay > maxDelay {
			delay = maxDelay
		}
		time.Sleep(delay)
	}
}

func sendAllMessagesOnce(client *ClientManager, msgs []queuedMessage, transactionId string) error {
	for _, qm := range msgs {
		qmMsgBytes, err := json.Marshal(qm)
		if err != nil {
			logger.Error("Failed to marshal qm message",
				zap.String("transactionId", transactionId),
				zap.String("status", "error"),
				zap.Error(err))
			return err
		}
		if err := client.conn.WriteMessage(websocket.TextMessage, qmMsgBytes); err != nil {
			logger.Error("Failed to send message",
				zap.String("transactionId", transactionId),
				zap.String("status", "error"),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *Server) Run(addr string) {
	http.HandleFunc("/ws", s.HandleClient)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func main() {
	defer logger.Sync()

	logger.Info("Application started",
		zap.String("transactionId", utils.GenerateTransactionID()),
		zap.String("status", "begin"),
		zap.String("version", "1.0.0"),
		zap.Time("timestamp", time.Now()),
	)

	server := NewServer()

	server.topics["news"] = NewTopic("news")
	server.topics["sports"] = NewTopic("sports")
	server.topics["tech"] = NewTopic("tech")

	logger.Info("Server starting",
		zap.String("transactionId", utils.GenerateTransactionID()),
		zap.String("status", "begin"),
		zap.String("addr", ":8082"))
	server.Run(":8082")
	logger.Info("Server stopped",
		zap.String("transactionId", utils.GenerateTransactionID()),
		zap.String("status", "finish"))

}
