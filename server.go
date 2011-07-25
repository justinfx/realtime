package main

/*
	Server
	
	Stores a reference to the current socketio server instance
	and handles all communications between connected clients.
*/


import (
	"os"
	"socketio"
	"json"
	//"tideland-rdc.googlecode.com/hg"
)


type ServerHandler struct {
	sio *socketio.SocketIO 
	//db	*rdc.RedisDatabase
}

func (s *ServerHandler) OnConnect(c *socketio.Conn) {
	Debugln("New connection:", c)
}

func (s *ServerHandler) OnDisconnect(c *socketio.Conn) {
	Debugln("Client Disconnected:", c)
}

// When a raw message comes in from a connected client, we need
// to parse it and determine what kind it is and how to route it.
func (s *ServerHandler) OnMessage(c *socketio.Conn, data socketio.Message) {
	Debugln("Incoming message from client:", c.String())
	
	
	// first try to see if the data is recognized as valid JSON
	raw, ok := data.JSON(); 
	if !ok {
		raw = data.Data()
	}
	Debugln("Raw message from client:", raw)
	
	msg := NewMessage()
	err := json.Unmarshal([]byte(raw), msg)
	if err != nil {
		Debugln(err, "JSON:", raw, "MSG:", data.Data())
		return 
	}
	msg.raw = raw

	switch msg.Type {
	
	case "command": 
		msg.mtype = CommandType
		err = s.handleCommand(c, msg)
	
	case "message": 
		msg.mtype = MessageType
		err = s.handleMessage(c, msg)
	
	default: 
		err = os.NewError("Malformed command message")	
	}
	
	if err != nil {
		Debugln("Errors during message handling:", err)
	}
	
}

// The message was a 'command' type
// If its a system command, route it to the handler
// otherwise, forward it to the other clients as a generic message
func (s *ServerHandler) handleCommand(c *socketio.Conn, msg *message) (err os.Error) {

	switch msg.Data["command"].(string) {
	
	case "": err = os.NewError("Malformed command message")	
	case "subscribe": err = s.subscribeCmd(c, msg) 
	case "unsubscribe": err = s.unsubscribeCmd(c, msg) 
	case "init": err = s.initCmd(c, msg) 
	
	default: 
		// not a system command. forward it on
		if msg.Channel != "" {
			Debugln("Forwarding generic command message:", msg.raw)
			s.sio.Broadcast(msg.raw)
		} else {
			err = os.NewError("Generic command message has no channel")
		}		
	}	

	return err
	
}

// Raw message was a 'message' type. Publish this
// to clients on the same channel
func (s *ServerHandler) handleMessage(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("msgHandler():", c, msg.raw)
	s.sio.Broadcast(msg)
	return err
}

func (s *ServerHandler) publish(msg *message) (err os.Error) {
	if msg.Channel == "" || msg.Data == nil || len(msg.Data) == 0 {
		err = os.NewError("msg either has no channel or no data. not publishing")
		return err
	}
	
	s.db.Publish(msg.Channel, msg)
	
	return 
}

//
// System Commands
//
func (s *ServerHandler) subscribeCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("subscribeCmd():", c, msg.raw)
	reply := NewCommand()
	reply.Data["command"] = "onSubscribe"
	reply.Data["options"] = msg.Data["options"]
	s.sio.BroadcastExcept(c, reply)
	return err
}

func (s *ServerHandler) unsubscribeCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("unsubscribeCmd():", c, msg.raw)
	reply := NewCommand()
	reply.Data["command"] = "onUnsubscribe"
	reply.Data["options"] = msg.Data["options"]
	s.sio.BroadcastExcept(c, reply)
	return err
}

func (s *ServerHandler) initCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("initCmd():", c, msg.raw)
	
	return
}
// END System Commands ----]
