package main

import (
	"time"
	
	// 3rd party
	"socketio"
)

const (
	CommandType = iota
	MessageType
)


/*
	Message

	Realtime-specific message structure.
	Raw socket.io messages are conformed into this
	for internal use and then passing back to the
	client.
*/
type message struct {
	Type      string                 "type"
	Channel   string                 "channel"
	Success   bool                   "success"
	Error     string                 "error"
	Identity  string                 "identity"
	Timestamp string                 "timestamp"
	Data      map[string]interface{} "data" // client-side specific

	raw   string
	mtype int
	conn  *socketio.Conn
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
		Timestamp: time.UTC().String(),
		Data:      map[string]interface{}{},
		mtype:     CommandType,
	}
}

func NewMessage() *message {
	return &message{
		Type:      "message",
		Success:   true,
		Timestamp: time.UTC().String(),
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
