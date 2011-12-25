package main

/*
	Server

	Stores a reference to the current socketio server instance
	and handles all communications between connected clients.
*/

import (
	"errors"
	"fmt"

	"sync"

	// 3rd party
	"socketio"
)

type ServerHandler struct {
	Sio *socketio.SocketIO

	subs    map[string][]*Client
	idents  map[string]*Client
	clients map[string]*Client

	msgChannel  chan *DispatchReq
	srvcChannel chan *DispatchReq
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

		msgChannel:  make(chan *DispatchReq, 5000),
		srvcChannel: make(chan *DispatchReq, 500),
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

	defer func() {
		if r := recover(); r != nil {
			Debugln("OnDisconnect(): Recovered from disconnecting client:", r)
			return
		}
	}()

	s.clientsLock.RLock()
	client, ok := s.clients[c.String()]
	s.clientsLock.RUnlock()

	if ok {

		client.lock.RLock()
		identity := client.Identity

		wg := &sync.WaitGroup{}

		if len(client.Conns) <= 1 {
			Debugln("OnDisconnect(): Client is last in group. Unsubscribing", client.Channels)

			msgs := []*message{}
			for _, val := range client.Channels {
				msg := NewCommand()
				msg.Channel = val
				msg.Data["command"] = "unsubscribe"
				msg.Identity = identity
				msgs = append(msgs, msg)
			}
			client.lock.RUnlock()

			for _, m := range msgs {
				wg.Add(1)
				go func() {
					req := NewDispatchReq(c, m, true)
					s.unsubscribeCmd(req)
					wg.Done()
				}()
			}

			if identity != "" {
				s.identsLock.Lock()
				delete(s.idents, identity)
				s.identsLock.Unlock()
			}

		} else {
			client.lock.RUnlock()
		}

		wg.Wait()

		client.RemoveConn(c)
	}

	s.clientsLock.Lock()
	delete(s.clients, c.String())
	//	Debugln("OnDisconnect: cleared connection from client list")
	s.clientsLock.Unlock()

}

// When a raw message comes in from a connected client, we need
// to parse it and determine what kind it is and how to route it.
func (s *ServerHandler) OnMessage(c *socketio.Conn, data socketio.Message) {
	if s.quitting {
		return
	}

	// first try to see if the data is recognized as valid JSON
	raw, ok := data.JSON()
	if !ok {
		raw = data.Bytes()
	}
	Debugln("Raw message from client:", c.String(), string(raw))

	msg, err := NewJsonMessage(raw)
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
		err = errors.New("Malformed command message")
	}

	if err != nil {
		Debugln("Errors during message handling:", err)
	}

}

// The message was a 'command' type
// If its a system command, route it to the handler
// otherwise, forward it to the other clients as a generic message
func (s *ServerHandler) handleCommand(c *socketio.Conn, msg *message) (err error) {

	switch msg.Data["command"].(string) {

	case "":
		err = errors.New("Malformed command message")

	case "subscribe":
		s.subscribeCmd(NewDispatchReq(c, msg, false))

	case "unsubscribe":
		s.unsubscribeCmd(NewDispatchReq(c, msg, false))

	case "init":
		s.initCmd(NewDispatchReq(c, msg, false))

	default:
		// not a system command. forward it on
		if msg.Channel != "" {
			Debugln("Forwarding generic command message:", msg.raw)
			err = s.publish(c, msg)
		} else {
			err = errors.New("Generic command message has no channel")
		}
	}

	return err

}

// Raw message was a 'message' type. Publish this
// to clients on the same channel
func (s *ServerHandler) handleMessage(c *socketio.Conn, msg *message) (err error) {
	Debugln("msgHandler():", c, msg.raw)

	err = s.publish(c, msg)

	return err
}

func (s *ServerHandler) publish(c *socketio.Conn, msg *message) (err error) {

	if msg.Channel == "" || msg.Data == nil || len(msg.Data) == 0 {
		err = errors.New("msg either has no channel or no data. not publishing")
		return err
	}

	req := NewDispatchReq(c, msg, false)
	s.msgChannel <- req

	return
}

func (s *ServerHandler) subscribeCmd(req *DispatchReq) {
	Debugln("subscribeCmd():", req.Conn, req.Msg.raw)

	s.srvcChannel <- req
	if req.Wait {
		<-req.done
	}

	return
}

func (s *ServerHandler) unsubscribeCmd(req *DispatchReq) {
	Debugln("unsubscribeCmd():", req.Conn, req.Msg.raw)

	s.srvcChannel <- req
	if req.Wait {
		<-req.done
	}

	return
}

// A client should first send an init command to establish their
// identity, and optional batch subscribe to any channels
func (s *ServerHandler) initCmd(req *DispatchReq) {
	Debugln("initCmd():", req.Conn, req.Msg.raw)

	c := req.Conn
	msg := req.Msg

	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	var (
		client *Client
		ok     bool
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
			client.AddConn(c)
			Debugln("initCmd(): adding conn to existing Client group:", client)
		} else {
			client = &Client{
				Identity: msg.Identity,
				Conns:    []*socketio.Conn{c},
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

	for i := 0; i < 2; i++ {
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
		req     *DispatchReq
		msg     *message
		members []*Client
	)

	for req = range s.msgChannel {

		msg = req.Msg

		if msg.Channel == "" {
			req.SetDone()
			continue
		}

		members = s.subs[msg.Channel]
		if members == nil || len(members) == 0 {
			req.SetDone()
			continue
		}

		//Debugln("startDispatcher(): Sending message w/ data - ", msg.Data)

		for i, _ := range members {
			for j := 0; j < len(members[i].Conns); {
				if err := members[i].Conns[j].Send(msg); err != nil {
					members[i].Conns = append(members[i].Conns[:j], members[i].Conns[j+1:]...)
					//s.subs[msg.Channel] = members
				} else {
					j++
				}
			}
		}
		req.SetDone()
	}
	s.quit <- true
}

func (s *ServerHandler) dispatchServices() {
	// Service messages include subscribe/unsubscribe
	// commands and are checked on a seperate channel
	// from messages so that their queue doest get 
	// flooded

	var (
		req        *DispatchReq
		msg        *message
		reply      *message
		client     *Client
		clientTest *Client
		members    []*Client
		ok         bool
	)

Dispatch:
	for req = range s.srvcChannel {

		msg = req.Msg

		if msg.Channel == "" {
			req.SetDone()
			continue
		}

		members, ok = s.subs[msg.Channel]
		if !ok {
			members = []*Client{}
		}

		switch msg.Data["command"].(string) {

		case "subscribe":

			s.clientsLock.RLock()
			client = s.clients[req.Conn.String()]
			s.clientsLock.RUnlock()

			for _, clientTest = range members {
				if clientTest == client {
					err := errors.New("client already subscribed to channel")
					Debugln(err)
					req.SetDone()
					continue Dispatch
				}
			}
			members = append(members, client)
			s.subs[msg.Channel] = members

			reply = NewCommand()
			reply.Channel = msg.Channel
			reply.Identity = msg.Identity
			reply.Data["command"] = "onSubscribe"
			reply.Data["options"] = msg.Data["options"]
			reply.Data["count"] = len(members)

			s.publish(req.Conn, reply)

			client.AddChannel(msg.Channel)

			Debugf("dispatchServices(): subscribed %v => \"%v\"", client, msg.Channel)

		case "unsubscribe":

			ok = false

			s.clientsLock.RLock()
			client = s.clients[req.Conn.String()]
			s.clientsLock.RUnlock()

			for i := 0; i < len(members); {
				clientTest = members[i]
				//Debugf("dispatchServices(): %v == %v ? %v", clientTest, client, clientTest==client)
				if clientTest == client {
					members = append(members[:i], members[i+1:]...)
					s.subs[msg.Channel] = members
					ok = true
					Debugf("dispatchServices(): unsubscribing %v from %v", req.Conn, msg.Channel)
					break
				} else {
					i++
				}
			}

			if ok {
				reply = NewCommand()
				reply.Identity = msg.Identity
				reply.Channel = msg.Channel
				reply.Data["command"] = "onUnsubscribe"
				reply.Data["options"] = msg.Data["options"]
				reply.Data["count"] = len(s.subs[msg.Channel])

				s.publish(req.Conn, reply)

				client.RemoveChannel(msg.Channel)

			} else {
				err := errors.New("client was not subscribed to channel")
				Debugln(err)
				req.SetDone()
				continue
			}
		}
		req.SetDone()
	}
	s.quit <- true
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

func (c *Client) AddConn(conn *socketio.Conn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Conns = append(c.Conns, conn)
}

func (c *Client) RemoveConn(conn *socketio.Conn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := 0; i < len(c.Conns); {
		if c.Conns[i] == conn {
			c.Conns = append(c.Conns[:i], c.Conns[i+1:]...)
		} else {
			i++
		}
	}
}

func (c *Client) AddChannel(channel string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channels = append(c.Channels, channel)
}

func (c *Client) RemoveChannel(channel string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := 0; i < len(c.Channels); {
		if c.Channels[i] == channel {
			c.Channels = append(c.Channels[:i], c.Channels[i+1:]...)
		} else {
			i++
		}
	}
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
	Msg  *message
	Wait bool
	Conn *socketio.Conn
	done chan bool
}

func NewDispatchReq(c *socketio.Conn, m *message, wait bool) *DispatchReq {

	req := &DispatchReq{
		Msg:  m,
		Conn: c,
		Wait: wait,
	}

	if wait {
		req.done = make(chan bool)
	}

	return req
}

func (d *DispatchReq) SetDone() {
	if d.Wait {
		d.done <- true
		close(d.done)
	}
}
