package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/blackswarm/hive/internal/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // allow all origins for agents
}

// AgentHandler manages WebSocket connections to agents.
type AgentHandler struct {
	manager    *AgentManager
	logger     *slog.Logger
	taskQueues map[string]chan protocol.TaskMsg // agent ID → pending tasks
	mu         sync.Mutex
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(manager *AgentManager, logger *slog.Logger) *AgentHandler {
	return &AgentHandler{
		manager:    manager,
		logger:     logger,
		taskQueues: make(map[string]chan protocol.TaskMsg),
	}
}

// ServeHTTP handles a new WebSocket connection from an agent.
func (h *AgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Wait for registration message
	var msg protocol.Message
	if err := conn.ReadJSON(&msg); err != nil {
		h.logger.Error("failed to read register message", "error", err)
		return
	}
	if msg.Type != protocol.TypeRegister {
		h.logger.Warn("first message is not register", "type", msg.Type)
		return
	}

	var reg protocol.RegisterMsg
	if err := json.Unmarshal(msg.Payload, &reg); err != nil {
		h.logger.Error("failed to unmarshal register", "error", err)
		return
	}

	agentInfo := AgentInfo{
		ID:       reg.AgentID,
		Hostname: reg.Hostname,
		OS:       reg.OS,
		IP:       reg.IP,
		LastSeen: time.Now(),
		Status:   "online",
	}
	h.manager.Register(reg.AgentID, agentInfo, conn)
	defer h.manager.Unregister(reg.AgentID)

	h.logger.Info("agent registered", "id", reg.AgentID, "hostname", reg.Hostname)

	// Create a task queue for this agent
	taskQueue := make(chan protocol.TaskMsg, 16)
	h.mu.Lock()
	h.taskQueues[reg.AgentID] = taskQueue
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.taskQueues, reg.AgentID)
		h.mu.Unlock()
	}()

	// Start writer goroutine
	done := make(chan struct{})
	go h.writePump(conn, taskQueue, done)

	// Read pump: receive results from agent
	for {
		var msg protocol.Message
		if err := conn.ReadJSON(&msg); err != nil {
			h.logger.Info("agent disconnected", "id", reg.AgentID, "error", err)
			break
		}
		switch msg.Type {
		case protocol.TypeResult:
			var result protocol.ResultMsg
			if err := json.Unmarshal(msg.Payload, &result); err != nil {
				h.logger.Error("failed to unmarshal result", "error", err)
				continue
			}
			h.logger.Info("task result received", "task_id", result.TaskID, "agent", reg.AgentID)
			// Here you could store the result or notify a waiting operator.
		case protocol.TypeDestroy:
			h.logger.Info("agent self-destruct", "id", reg.AgentID)
			return
		default:
			h.logger.Warn("unknown message type from agent", "type", msg.Type)
		}
	}
	close(done)
}

// writePump sends tasks from the queue to the agent via WebSocket.
func (h *AgentHandler) writePump(conn *websocket.Conn, queue chan protocol.TaskMsg, done chan struct{}) {
	for {
		select {
		case task := <-queue:
			payload, _ := json.Marshal(task)
			msg := protocol.Message{
				Type:    protocol.TypeTask,
				Payload: payload,
			}
			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Error("failed to send task", "error", err)
				return
			}
		case <-done:
			return
		}
	}
}

// SendTask enqueues a task to be sent to a specific agent.
func (h *AgentHandler) SendTask(agentID string, task protocol.TaskMsg) error {
	h.mu.Lock()
	q, ok := h.taskQueues[agentID]
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("agent %s not connected", agentID)
	}
	select {
	case q <- task:
		return nil
	default:
		return fmt.Errorf("task queue for agent %s is full", agentID)
	}
}
