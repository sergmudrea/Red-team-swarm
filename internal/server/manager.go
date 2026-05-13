// Package server implements the hive command centre (WebSocket server, agent manager, reporting).
package server

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AgentInfo holds the publicly visible metadata of a connected agent.
type AgentInfo struct {
	ID       string    `json:"id"`
	Hostname string    `json:"hostname"`
	OS       string    `json:"os"`
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"` // "online", "offline", etc.
}

// agentEntry is the internal representation stored in the manager.
type agentEntry struct {
	info AgentInfo
	conn *websocket.Conn
}

// AgentManager provides thread‑safe storage for connected agents.
type AgentManager struct {
	agents sync.Map // id (string) → *agentEntry
}

// Register adds a new agent or updates an existing one.
func (m *AgentManager) Register(id string, info AgentInfo, conn *websocket.Conn) {
	m.agents.Store(id, &agentEntry{info: info, conn: conn})
}

// Unregister removes an agent from the manager.
func (m *AgentManager) Unregister(id string) {
	m.agents.Delete(id)
}

// GetAgent returns the agent info and its WebSocket connection, if found.
func (m *AgentManager) GetAgent(id string) (AgentInfo, *websocket.Conn, bool) {
	v, ok := m.agents.Load(id)
	if !ok {
		return AgentInfo{}, nil, false
	}
	entry := v.(*agentEntry)
	return entry.info, entry.conn, true
}

// ListAgents returns a snapshot of all registered agents.
func (m *AgentManager) ListAgents() []AgentInfo {
	var list []AgentInfo
	m.agents.Range(func(key, value any) bool {
		entry := value.(*agentEntry)
		list = append(list, entry.info)
		return true
	})
	return list
}
