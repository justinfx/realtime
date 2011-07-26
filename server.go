package main

/*
	Server

	Stores a reference to the current socketio server instance
	and handles all communications between connected clients.
*/


import (
	"os"
	"container/list"
	"socketio"
	"json"
	//"tideland-rdc.googlecode.com/hg"
)


type ServerHandler struct {
	Sio *socketio.SocketIO
	//db	*rdc.RedisDatabase

	subs        map[string]*list.List
	msgChannel  chan *message
	srvcChannel chan *message
}

type Client struct {
	Identity string
	Conn     *socketio.Conn
}

type DispatchReq struct {
	msg    *message
	result chan interface{}
}

func NewServerHandler(sio *socketio.SocketIO) (s *ServerHandler) {
	s = &ServerHandler{
		Sio:         sio,
		subs:        make(map[string]*list.List),
		msgChannel:  make(chan *message),
		srvcChannel: make(chan *message),
	}
	s.startDispatcher()

	return s
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
	raw, ok := data.JSON()
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

	case "":
		err = os.NewError("Malformed command message")

	case "subscribe":
		err = s.subscribeCmd(c, msg)

	case "unsubscribe":
		err = s.unsubscribeCmd(c, msg)

	case "init":
		err = s.initCmd(c, msg)

	default:
		// not a system command. forward it on
		if msg.Channel != "" {
			Debugln("Forwarding generic command message:", msg.raw)
			s.publish(msg)
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

	msg.conn = c
	s.publish(msg)

	return err
}

func (s *ServerHandler) publish(msg *message) (err os.Error) {

	if msg.Channel == "" || msg.Data == nil || len(msg.Data) == 0 {
		err = os.NewError("msg either has no channel or no data. not publishing")
		return err
	}

	go func() {
		s.msgChannel <- msg
	}()

	return
}


func (s *ServerHandler) subscribeCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("subscribeCmd():", c, msg.raw)

	msg.conn = c
	go func() {
		s.srvcChannel <- msg
	}()

	return err
}

func (s *ServerHandler) unsubscribeCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("unsubscribeCmd():", c, msg.raw)

	msg.conn = c
	go func() {
		s.srvcChannel <- msg
	}()

	return err
}

func (s *ServerHandler) initCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("initCmd():", c, msg.raw)

	return
}


// A goroutine that coordinates the internal message
// routing. Delivers messages to all members of the
// given channel. Also processes (un)subscription commands
// by updating an internal map.
func (s *ServerHandler) startDispatcher() {

	go func() {

		var (
			msg                *message
			reply              *message
			ok                 bool
			typ                string
			elem               *list.Element
			client, clientTest Client

			members  = list.New()
			toRemove = list.New()
		)

		for {

			select {

			// Service messages include subscribe/unsubscribe
			// commands and are checked on a seperate channel
			// from messages so that their queue doest get 
			// flooded
			case msg, ok = <-s.srvcChannel:
				if !ok {
					return
				}
				if msg.Channel == "" {
					continue
				}

				members = s.subs[msg.Channel]
				if members == nil {
					members = list.New()
					s.subs[msg.Channel] = members
				}

				typ = msg.Data["command"].(string)

				switch typ {

				// add this member to the given channel
				case "subscribe":
					client = Client{msg.Identity, msg.conn}

					for e := members.Front(); e != nil; e = e.Next() {
						clientTest = e.Value.(Client)
						if clientTest.Conn == client.Conn {
							err := os.NewError("client already subscribed to channel")
							Debugln(err)
							continue
						}
					}
					_ = members.PushBack(client)

					reply = NewCommand()
					reply.Data["command"] = "onSubscribe"
					reply.Data["options"] = msg.Data["options"]

					s.publish(reply)

					Debugf("startDispatcher(): subscribed client %v to \"%v\"", client, msg.Channel)

				// remove this member from the given channel
				case "unsubscribe":
					ok = false
					for e := members.Front(); e != nil; e = e.Next() {
						client = e.Value.(Client)
						if client.Conn == msg.conn {
							elem = e.Prev()
							members.Remove(e)
							ok = true
							Debugln("startDispatcher(): unsubscribing %v from %v", msg.conn, msg.Channel)

							break
						}
					}
					if ok {
						reply = NewCommand()
						reply.Data["command"] = "onUnsubscribe"
						reply.Data["options"] = msg.Data["options"]

						s.publish(reply)
					} else {
						err := os.NewError("client was not subscribed to channel")
						Debugln(err)
						continue
					}
				}

			// standard message channel. a message that will get
			// published to all other connections currently
			// subscribed to the same channel
			case msg, ok = <-s.msgChannel:
				if !ok {
					return
				}
				if msg.Channel == "" {
					continue
				}

				members = s.subs[msg.Channel]
				if members == nil {
					continue
				}

				Debugf("startDispatcher(): Routing msg from to \"%v\"", msg.Channel)

				toRemove.Init()

				for e := members.Front(); e != nil; e = e.Next() {

					client = e.Value.(Client)
					if client.Conn.Send(msg) != nil {
						// getting an error here most likely means
						// that the client disconnected. we should not
						// track their subscriptions
						// TODO: impliment a form of persistance so that
						// reconnecting clients can resume the last state
						toRemove.PushBack(e)
					}
				}
				if toRemove.Len() > 0 {
					for e := toRemove.Front(); e != nil; e = e.Next() {
						elem = e.Value.(*list.Element)
						members.Remove(elem)
					}
					toRemove.Init()
				}

			}
		}
	}()
}
