package hub

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

// Terminal message types
const (
	TerminalMsgInput  = "input"
	TerminalMsgResize = "resize"
)

// TerminalMessage represents a message from the client
type TerminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// WebSocketTerminal bridges WebSocket and k8s exec streams
type WebSocketTerminal struct {
	conn   *websocket.Conn
	sizeCh chan remotecommand.TerminalSize
	doneCh chan struct{}
	mu     sync.Mutex
}

// NewWebSocketTerminal creates a new terminal bridge
func NewWebSocketTerminal(conn *websocket.Conn) *WebSocketTerminal {
	return &WebSocketTerminal{
		conn:   conn,
		sizeCh: make(chan remotecommand.TerminalSize, 1),
		doneCh: make(chan struct{}),
	}
}

// Next implements remotecommand.TerminalSizeQueue
func (t *WebSocketTerminal) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeCh:
		return &size
	case <-t.doneCh:
		return nil
	}
}

// Read implements io.Reader - reads from WebSocket
func (t *WebSocketTerminal) Read(p []byte) (int, error) {
	_, message, err := t.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	var msg TerminalMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		// Treat as raw input if not JSON
		return copy(p, message), nil
	}

	switch msg.Type {
	case TerminalMsgResize:
		select {
		case t.sizeCh <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}:
		default:
			// Non-blocking send, drop resize if channel is full
		}
		// Return empty read for resize messages - don't pass to stdin
		return 0, nil
	case TerminalMsgInput:
		return copy(p, []byte(msg.Data)), nil
	default:
		return copy(p, message), nil
	}
}

// Write implements io.Writer - writes to WebSocket
func (t *WebSocketTerminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes the terminal session
func (t *WebSocketTerminal) Close() {
	select {
	case <-t.doneCh:
		// Already closed
	default:
		close(t.doneCh)
	}
}

// Ensure interfaces are implemented
var _ io.Reader = (*WebSocketTerminal)(nil)
var _ io.Writer = (*WebSocketTerminal)(nil)
var _ remotecommand.TerminalSizeQueue = (*WebSocketTerminal)(nil)
