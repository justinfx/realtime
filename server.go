package main

/*
	Server

	Stores a reference to the current socketio server instance
	and handles all communications between connected clients.
*/


import (
	"os"
	"json"
	"sync"
	"fmt"

	// 3rd party
	"github.com/justinfx/go-socket.io"
	//"github.com/madari/go-socket.io"
	//"socketio" // dev only
)


type ServerHandler struct {
	Sio *socketio.SocketIO

	subs        map[string][]*Client
	idents      map[string]*Client
	clients     map[string]*Client
	
	msgChannel  chan *message
	srvcChannel chan *message
	quit        chan bool
	quitting    bool

	identsLock, clientsLock sync.RWMutex
}


func NewServerHandler(sio *socketio.SocketIO) (s *ServerHandler) {

	s = &ServerHandler{
		Sio: sio,

		subs:    make(map[string][]*Client),
		idents:  make(map[string]*Client),
		clients: make(map[string]*Client),

		msgChannel:  make(chan *message, 5000),
		srvcChannel: make(chan *message, 500),
		quit:        make(chan bool),
		quitting:    false,
	}

	go s.dispatchServices()
	go s.dispatchMessages()

	return s
}

// When a new user connects, associate their connection
// with an id -> Client object
func (s *ServerHandler) OnConnect(c *socketio.Conn) {

	/*
	s.clientsLock.Lock()
	s.clients[c.String()] = &Client{Conns: c}
	s.clientsLock.Unlock()
	*/
}

// When a client disconnected, remove their Client
// object reference
func (s *ServerHandler) OnDisconnect(c *socketio.Conn) {

	s.clientsLock.Lock()
	client := s.clients[c.String()]
	s.clientsLock.Unlock()

	msgs := []*message{}

	client.lock.RLock()
	for _, val := range client.Channels {
		msg := NewCommand()
		msg.Channel = val
		msg.Data["command"] = "unsubscribe"
		msg.Identity = client.Identity
		msgs = append(msgs, msg)
	}
	client.lock.RUnlock()

	for _, m := range msgs {
		s.unsubscribeCmd(c, m)
	}

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

	msg, err := s.jsonToData(raw)
	if err != nil {
		Debugln(err, "JSON:", raw, "MSG:", data.Data())
		return
	}

	s.clientsLock.RLock()
	client, ok := s.clients[c.String()]

	if !ok || !client.HasInit() {
		if msg.Type == "command" && msg.Data["command"] == "init" {
			//yay. we want an init command here
		} else {
			errMsg := NewErrorMessage("Client has not sent init command yet!")
			Debugln(errMsg)
			c.Send(errMsg)
			s.clientsLock.RUnlock()
			return
		}	
	} else if ok {
		msg.Identity = client.Identity
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
			err = s.publish(msg)
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
	err = s.publish(msg)

	return err
}

func (s *ServerHandler) publish(msg *message) (err os.Error) {

	if msg.Channel == "" || msg.Data == nil || len(msg.Data) == 0 {
		err = os.NewError("msg either has no channel or no data. not publishing")
		return err
	}

	s.msgChannel <- msg

	return
}


func (s *ServerHandler) subscribeCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("subscribeCmd():", c, msg.raw)

	msg.conn = c
	s.srvcChannel <- msg

	return err
}

func (s *ServerHandler) unsubscribeCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("unsubscribeCmd():", c, msg.raw)

	msg.conn = c
	s.srvcChannel <- msg

	return err
}

// A client should first send an init command to establish their
// identity, and optional batch subscribe to any channels
func (s *ServerHandler) initCmd(c *socketio.Conn, msg *message) (err os.Error) {
	Debugln("initCmd():", c, msg.raw)

	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	var (
		client *Client
		ok bool
	)
	
	client, ok = s.clients[c.String()]
	if ok && client.HasInit() {
		Debugln("initCmd(): Client has already init before:", c)
		return
	} 

	if msg.Identity != "" {
		s.identsLock.Lock()
		client, ok = s.idents[msg.Identity]
		if ok {
			client.Conns = append(client.Conns, c)
			Debugln("initCmd(): adding conn to existing Client group:", client)
		} else {
			client = &Client{
				Identity: msg.Identity,
				Conns: []*socketio.Conn{c},
			}
			Debugln("initCmd(): conn is new. creating new Client group:", client)
		}
		s.idents[msg.Identity] = client
		s.identsLock.Unlock()
		
		s.clients[c.String()] = client
		
	} else {
		client = &Client{
			Conns: []*socketio.Conn{c},
		}
		s.clients[c.String()] = client
	}

	/*
	// TODO: FIXME
	// Need to properly unbox the []string. 
	// Right now this crashes
	channels := msg.Data["channels"]
	if channels != nil {
		for _, channel := range channels.([]string) {
			cmdMsg := NewCommand()
			cmdMsg.Channel = channel
			s.subscribeCmd(c, cmdMsg)
		}
	}
	*/

	client.SetInit(true)

	return
}


func (s *ServerHandler) Shutdown() {
	s.quitting = true

	close(s.srvcChannel)
	close(s.msgChannel)

	for i:=0; i < 2; i++ {
		<-s.quit
	}
}

// A goroutine that coordinates the internal message
// routing. Delivers messages to all members of the
// given channel. Also processes (un)subscription commands
// by updating an internal map.
func (s *ServerHandler) dispatchMessages() {

	// standard message channel. a message that will get
	// published to all other connections currently
	// subscribed to the same channel
	
	var (
		msg *message
		ok bool
		members    []*Client
	)
	
	for {
		select {
		case msg, ok = <-s.msgChannel:
			if !ok {
				s.quit <- true
				return
			}
			if msg.Channel == "" {
				continue
			}

			members = s.subs[msg.Channel]
			if members == nil || len(members) == 0 {
				continue
			}

			//Debugln("startDispatcher(): Sending message w/ data - ", msg.Data)

			for i, _ := range members {
				for j := 0; j < len(members[i].Conns); {
					if err :=  members[i].Conns[j].Send(msg); err != nil {
						members[i].Conns = append(members[i].Conns[:j], members[i].Conns[j+1:]...)
						//s.subs[msg.Channel] = members
					} else {
						j++
					}
				}
			}
		}
	}
}

func (s *ServerHandler) dispatchServices() {
	// Service messages include subscribe/unsubscribe
	// commands and are checked on a seperate channel
	// from messages so that their queue doest get 
	// flooded
	
	var (
		msg        *message
		reply      *message
		ok         bool
		client     *Client
		clientTest *Client
		members    []*Client
	)

Dispatch:
	for {
		select {
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
				members = []*Client{}
			}

			switch msg.Data["command"].(string) {

			case "subscribe":

				s.clientsLock.RLock()
				client = s.clients[msg.conn.String()]
				s.clientsLock.RUnlock()

				for _, clientTest = range members {
					if clientTest == client {
						err := os.NewError("client already subscribed to channel")
						Debugln(err)
						continue Dispatch
					}
				}
				members = append(members, client)
				s.subs[msg.Channel] = members

				reply = NewCommand()
				reply.Channel = msg.Channel
				reply.Identity = client.Identity
				reply.Data["command"] = "onSubscribe"
				reply.Data["options"] = msg.Data["options"]
				reply.Data["count"] = len(members)

				s.publish(reply)

				client.lock.Lock()
				client.Channels = append(client.Channels, msg.Channel)
				client.lock.Unlock()

				Debugf("dispatchServices(): subscribed %v => \"%v\"", client, msg.Channel)

			case "unsubscribe":

				ok = false

				s.clientsLock.RLock()
				client = s.clients[msg.conn.String()]
				s.clientsLock.RUnlock()
				
				for i := 0; i < len(members); {
					clientTest = members[i]
					if clientTest == client {
						members = append(members[:i], members[i+1:]...)
						ok = true
						Debugf("dispatchServices(): unsubscribing %v from %v", msg.conn, msg.Channel)
						break
					} else {
						i++
					}
				}

				if ok {
					s.subs[msg.Channel] = members

					reply = NewCommand()
					client.lock.RLock()
					reply.Identity = client.Identity
					client.lock.RUnlock()
					reply.Channel = msg.Channel
					reply.Data["command"] = "onUnsubscribe"
					reply.Data["options"] = msg.Data["options"]
					reply.Data["count"] = len(members)

					s.publish(reply)

					client.lock.Lock()
					for i, val := range client.Channels {
						if msg.Channel == val {
							client.Channels = append(client.Channels[:i], client.Channels[i+1:]...)
							break
						}
					}
					client.lock.Unlock()

				} else {
					err := os.NewError("client was not subscribed to channel")
					Debugln(err)
					continue
				}
			}
		}
	}
}

func (s *ServerHandler) jsonToData(raw string) (msg *message, err os.Error) {
	msg = NewMessage()
	err = json.Unmarshal([]byte(raw), msg)
	if err != nil {
		return nil, err
	}
	msg.raw = raw

	return msg, err
}


type Client struct {
	Identity string
	Conns    []*socketio.Conn
	Channels []string
	hasInit  bool
	lock     sync.RWMutex
}


func (c *Client) String() string {
	return fmt.Sprintf("Client{Identity: %v, #Conn: %d, Conn: %v}", c.Identity, len(c.Conns), c.Conns)
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
