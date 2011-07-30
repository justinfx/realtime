/*
	Server

	Stores a reference to the current socketio server instance
	and handles all communications between connected clients.
*/

package main


import (
	"os"
	"container/list"
	"json"
	"time"
	"sync"
	"fmt"

	// 3rd party
	"socketio"
	//"tideland-rdc.googlecode.com/hg"
)


type ServerHandler struct {
	Sio *socketio.SocketIO
	//db	*rdc.RedisDatabase

	subs, idents map[string]*list.List
	clients      map[string]*Client
	msgChannel   chan *message
	srvcChannel  chan *message
	quit         chan bool
	quitting     bool

	identsLock, clientsLock sync.RWMutex
}


func NewServerHandler(sio *socketio.SocketIO) (s *ServerHandler) {

	s = &ServerHandler{
		Sio: sio,

		subs:    make(map[string]*list.List),
		idents:  make(map[string]*list.List),
		clients: make(map[string]*Client),

		msgChannel:  make(chan *message),
		srvcChannel: make(chan *message),
		quit:        make(chan bool, 1),
		quitting:    false,
	}

	s.startDispatcher()

	return s
}

// When a new user connects, associate their connection
// with an id -> Client object
func (s *ServerHandler) OnConnect(c *socketio.Conn) {
	//Debugln("New connection:", c)

	s.clientsLock.Lock()
	s.clients[c.String()] = &Client{Conn: c}
	s.clientsLock.Unlock()
}

// When a client disconnected, remove their Client
// object reference
func (s *ServerHandler) OnDisconnect(c *socketio.Conn) {
	//Debugln("Client Disconnected:", c)

	s.clientsLock.Lock()
	s.clients[c.String()] = nil, false
	s.clientsLock.Unlock()
}

// When a raw message comes in from a connected client, we need
// to parse it and determine what kind it is and how to route it.
func (s *ServerHandler) OnMessage(c *socketio.Conn, data socketio.Message) {
	if s.quitting {
		return
	}

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

	s.clientsLock.RLock()
	client, ok := s.clients[c.String()]

	if ok {
		if !client.HasInit() {
			if msg.Type == "command" && msg.Data["command"] == "init" {
				//yay. we want an init command here
			} else {
				errMsg := NewErrorMessage("Client has not sent init command yet!")
				Debugln(errMsg)
				c.Send(errMsg)
				s.clientsLock.RUnlock()
				return
			}
		} else {
			// for anything other than the init, we want to keep passing
			// the original Identity value with messages.
			msg.Identity = client.Identity
		}
	} else {
		Debugln("Received msg from client, yet there is no Client object record from the connection")
		return
	}
	s.clientsLock.RUnlock()

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

// A client should first send an init command to establish their
// identity, and optional batch subscribe to any channels
func (s *ServerHandler) initCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("initCmd():", c, msg.raw)

	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	client := s.clients[c.String()]

	if client.HasInit() {
		return
	}

	if msg.Identity != "" {
		client.Identity = msg.Identity

		s.identsLock.Lock()
		idents := s.idents[client.Identity]
		if idents == nil {
			idents = list.New()
			s.idents[client.Identity] = idents
		}
		idents.PushBack(client)

		s.identsLock.Unlock()
	}

	channels := msg.Data["channels"]
	if channels != nil {
		for _, channel := range channels.([]string) {
			cmdMsg := NewCommand()
			cmdMsg.Channel = channel
			s.subscribeCmd(c, cmdMsg)
		}
	}

	client.SetInit(true)

	return
}


func (s *ServerHandler) Shutdown() {
	s.quitting = true
	// if we are in debug mode, just shutdown right away
	if !CONFIG.DEBUG {
		time.Sleep(5e9)
	}
	close(s.srvcChannel)
	close(s.msgChannel)
	<-s.quit
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
			client, clientTest *Client

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
					s.quit <- true
					return
				}
				if msg.Channel == "" {
					continue
				}

				members, ok = s.subs[msg.Channel]
				if !ok {
					members = list.New()
					s.subs[msg.Channel] = members
				}

				typ = msg.Data["command"].(string)

				switch typ {

				// add this member to the given channel
				case "subscribe":

					s.clientsLock.RLock()
					client = s.clients[msg.conn.String()]
					s.clientsLock.RUnlock()

					for e := members.Front(); e != nil; e = e.Next() {
						clientTest = e.Value.(*Client)
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

					Debugf("startDispatcher(): subscribed %v => \"%v\"", client, msg.Channel)

				// remove this member from the given channel
				case "unsubscribe":
					ok = false
					for e := members.Front(); e != nil; e = e.Next() {
						client = e.Value.(*Client)
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
					s.quit <- true
					return
				}
				if msg.Channel == "" {
					continue
				}

				members = s.subs[msg.Channel]
				if members == nil {
					continue
				}

				//Debugf("startDispatcher(): Routing msg to \"%v\"", msg.Channel)

				toRemove.Init()

				for e := members.Front(); e != nil; e = e.Next() {

					client = e.Value.(*Client)
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


type Client struct {
	Identity string
	Conn     *socketio.Conn
	hasInit  bool
	lock     sync.RWMutex
}

func (c *Client) String() string {
	return fmt.Sprintf("Client{Identity: %v, Conn: %v}", c.Identity, c.Conn.String())
}

func (c *Client) HasInit() bool {
	c.lock.RLock()
	init := c.hasInit
	c.lock.RUnlock()
	return init
}

func (c *Client) SetInit(val bool) {
	c.lock.Lock()
	c.hasInit = val
	c.lock.Unlock()
}

type DispatchReq struct {
	msg    *message
	result chan interface{}
}
