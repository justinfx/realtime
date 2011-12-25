package main

/*
	Message

	Realtime-specific message structure.
	Raw socket.io messages are conformed into this
	for internal use and then passing back to the
	client.
*/

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	CommandType = iota
	MessageType
)

type message struct {
	Type      string                 `json:"type"`
	Channel   string                 `json:"channel"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error"`
	Identity  string                 `json:"identity"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"` // client-side specific

	raw   string
	mtype int
}

func (m *message) String() string {
	return fmt.Sprintf("message{Type: %v, Channel: \"%v\", Error: \"%v\", Identity: %v, raw: \"%v\"}",
		m.Type, m.Channel, m.Error, m.Identity, m.raw)
}

func (m *message) setRaw(data string) {
	m.raw = data
}

func (m *message) getRaw() string {
	return m.raw
}

func NewCommand() *message {
	return &message{
		Type:      "command",
		Success:   true,
		Timestamp: time.Now().UTC().String(),
		Data:      map[string]interface{}{},
		mtype:     CommandType,
	}
}

func NewMessage() *message {
	return &message{
		Type:      "message",
		Success:   true,
		Timestamp: time.Now().UTC().String(),
		Data:      map[string]interface{}{},
		mtype:     MessageType,
	}
}

func NewErrorMessage(err string) *message {
	msg := NewMessage()
	msg.Error = err
	msg.Success = false
	return msg
}

func NewJsonMessage(raw []byte) (msg *message, err error) {
	msg = NewMessage()
	err = json.Unmarshal(raw, msg)
	if err != nil {
		return nil, err
	}
	msg.raw = string(raw)

	return msg, err
}
