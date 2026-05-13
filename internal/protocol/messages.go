// Package protocol defines JSON message types exchanged between agent and server.
package protocol

// Message types used in the "type" field of a message wrapper.
const (
	TypeRegister = "register"
	TypeTask     = "task"
	TypeResult   = "result"
	TypeDestroy  = "destroy"
	TypeProxyOn  = "proxy_on"
	TypeProxyOff = "proxy_off"
)

// Message is a generic envelope for all WebSocket communication.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// RegisterMsg is sent by an agent immediately after connecting.
type RegisterMsg struct {
	AgentID  string `json:"agent_id"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	IP       string `json:"ip"`
}

// TaskMsg is sent by the server to instruct an agent.
type TaskMsg struct {
	TaskID  string `json:"task_id"`
	Command string `json:"command"` // shell command to execute
	Timeout int    `json:"timeout"` // seconds, 0 means default
}

// ResultMsg is sent by the agent after executing a task.
type ResultMsg struct {
	TaskID string `json:"task_id"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error,omitempty"` // non-empty if execution itself failed
}
