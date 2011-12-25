package main

import (
	"fmt"
	"net/http"
	"socketio"
	"sync"
	"testing"
	"time"
)

const (
	SERVER_ADDR  = "localhost"
	PORT         = "9999"
	IDENT        = "TEST1"
	NUM_MSGS     = 1000
	eventConnect = iota
	eventDisconnect
	eventMessage
	eventCrash
)

var EVENTS chan *event
var _SERVER *ServerHandler

type event struct {
	conn      *socketio.Conn
	eventType uint8
	msg       socketio.Message
}

func newInit() *message {
	cmd := NewCommand()
	cmd.Data["command"] = "init"
	cmd.Identity = IDENT
	return cmd
}

func newSub() *message {
	cmd := NewCommand()
	cmd.Data["command"] = "subscribe"
	cmd.Channel = "chat"
	return cmd
}

func newUnsub() *message {
	cmd := NewCommand()
	cmd.Data["command"] = "unsubscribe"
	cmd.Channel = "chat"
	return cmd
}

func newMsg() *message {
	msg := NewMessage()
	msg.Channel = "chat"
	return msg
}

func newInitStr() string {
	return `{"type":"command","identity": "TEST1","data":{"command":"init"}}`
}
func newSubStr() string {
	return `{"type":"command","identity": "TEST1","channel": "chat","data":{"command":"subscribe"}}`
}
func newUnsubStr() string {
	return `{"type":"command","identity": "TEST1","channel": "chat","data":{"command":"unsubscribe"}}`
}
func newMsgStr(msg string) string {
	return fmt.Sprintf(`{"type":"message","identity": "TEST1","channel": "chat","data":{"msg":"%v"}}`, msg)
}

func startServer() <-chan *event {

	if _SERVER != nil && EVENTS != nil {
		return EVENTS
	}

	config := socketio.DefaultConfig
	config.QueueLength = 1000
	config.Resource = "/realtime/"
	config.ReconnectTimeout = 1e9
	config.Origins = []string{"*"}

	EVENTS = make(chan *event, 100)

	sio := socketio.NewSocketIO(&config)
	_SERVER = NewServerHandler(sio)

	sio.OnConnect(func(c *socketio.Conn) {
		_SERVER.OnConnect(c)
		EVENTS <- &event{c, eventConnect, nil}
	})
	sio.OnDisconnect(func(c *socketio.Conn) {
		_SERVER.OnDisconnect(c)
		EVENTS <- &event{c, eventDisconnect, nil}
	})
	sio.OnMessage(func(c *socketio.Conn, msg socketio.Message) {
		_SERVER.OnMessage(c, msg)
		EVENTS <- &event{c, eventMessage, msg}
	})
	go func() {
		http.ListenAndServe(fmt.Sprintf("%s:%s", SERVER_ADDR, PORT), sio.ServeMux())
		EVENTS <- &event{nil, eventCrash, nil}
	}()

	return EVENTS
}

func connectClient(t *testing.T) (*socketio.WebsocketClient, chan *message, chan bool) {

	// license test
	client := socketio.NewWebsocketClient(socketio.SIOCodec{})
	err := client.Dial("ws://127.0.0.1:"+PORT+"/realtime/websocket", "http://"+SERVER_ADDR+":"+PORT+"/")
	if err == nil {
		t.Fatal("Expected a license failure when connecting to 127.0.0.1")
	}
	client.Close()

	// really connect	
	clientMessage := make(chan *message)
	clientDisconnect := make(chan bool)

	client = socketio.NewWebsocketClient(socketio.SIOCodec{})

	client.OnMessage(func(msg socketio.Message) {
		j, _ := msg.JSON()
		obj, err := NewJsonMessage(j)
		if err != nil {
			t.Fatalf("Client received a message that was not valid JSON: %v, error: %v", string(j), err)
		}
		clientMessage <- obj
	})

	client.OnDisconnect(func() {
		clientDisconnect <- true
	})

	err = client.Dial("ws://"+SERVER_ADDR+":"+PORT+"/realtime/websocket", "http://"+SERVER_ADDR+":"+PORT+"/")
	if err != nil {
		t.Fatal(err)
	}

	// expect connection
	serverEvent := <-EVENTS
	if serverEvent.eventType != eventConnect {
		t.Fatalf("Expected eventConnect but got %v", serverEvent)
	}

	// init
	if err = client.Send(newInitStr()); err != nil {
		t.Fatal("Send init:", err)
	}
	serverEvent = <-EVENTS
	if serverEvent.eventType != eventMessage {
		t.Fatalf("Expected eventMessage but got %v", serverEvent)
	}

	// subscribe
	if err = client.Send(newSubStr()); err != nil {
		t.Fatal("Send subscribe:", err)
	}
	serverEvent = <-EVENTS
	if serverEvent.eventType != eventMessage {
		t.Fatalf("Expected eventMessage but got %v", serverEvent)
	}
	msg := <-clientMessage
	if msg.Data["count"].(float64) != 1 {
		t.Fatalf("Expected subscription count to be 1 but got %v", msg.Data["count"])
	}

	return client, clientMessage, clientDisconnect
}

// TestMessages
// Starts a server routine, and sends messages.
// Checks that the same number of messages are returned
// with proper data values.
func TestMessages(t *testing.T) {

	CONFIG.DEBUG = false

	serverEvents := startServer()
	client, clientMessage, clientDisconnect := connectClient(t)

	t.Logf("Sending and receiving %d messages", NUM_MSGS)

	iook := new(sync.WaitGroup)

	time.Sleep(1e9)

	iook.Add(1)
	go func() {
		for i := 0; i < NUM_MSGS; i++ {
			val := fmt.Sprintf("%d", i)
			msg := newMsgStr(val)

			if err := client.Send(msg); err != nil {
				t.Fatal("Send:", err)
			}
		}

		iook.Done()
	}()

	iook.Add(1)
	go func() {
		for j := 0; j < NUM_MSGS; j++ {
			reply := <-clientMessage

			// message data check
			val := fmt.Sprintf("%d", j)
			if reply.Data["msg"] != val {
				t.Fatalf("Expected %v but got %v", val, reply.Data["msg"])
			}

			// identity should always get passed around
			if reply.Identity != IDENT {
				t.Fatalf("Expected idenity to be %v but got %v", IDENT, reply.Identity)
			}
		}

		iook.Done()
	}()

	iook.Add(1)
	go func() {
		for k := 0; k < NUM_MSGS; k++ {
			serverEvent := <-serverEvents
			if serverEvent.eventType != eventMessage {
				t.Fatalf("Expected eventMessage but got %v", serverEvent)
			}
		}
		iook.Done()
	}()

	iook.Wait()

	// unsubscribe
	if err := client.Send(newUnsubStr()); err != nil {
		t.Fatal("Send unsubscribe:", err)
	}
	serverEvent := <-EVENTS
	if serverEvent.eventType != eventMessage {
		t.Fatalf("Expected eventMessage but got %v", serverEvent)
	}

	go func() {
		if err := client.Close(); err != nil {
			t.Fatal("Close:", err)
		}
	}()

	t.Log("Waiting for client disconnect")
	<-clientDisconnect

	t.Log("Waiting for server disconnect")
	serverEvent = <-EVENTS
	if serverEvent.eventType != eventDisconnect {
		t.Fatalf("Expected disconnect event, but got %q", serverEvent)
	}

	_SERVER.Shutdown()

	CONFIG.DEBUG = false
}
