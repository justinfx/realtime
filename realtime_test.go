package main

import (
	//"socketio"
	"github.com/justinfx/go-socket.io"
	"fmt"
	"http"
	"time"
	"testing"
	"os"
)

const (
	SERVER_ADDR = "localhost:9999"
	IDENT       = "TEST1"

	eventConnect = iota
	eventDisconnect
	eventMessage
	eventCrash
)

var EVENTS chan *event
var SERVER *ServerHandler


type event struct {
	conn      *socketio.Conn
	eventType uint8
	msg       socketio.Message
}


func startServer() <-chan *event {

	if SERVER != nil && EVENTS != nil {
		return EVENTS
	}

	config := socketio.DefaultConfig
	config.QueueLength = 50
	config.Resource = "/realtime/"
	config.ReconnectTimeout = 2e9
	config.Origins = []string{"*"}

	EVENTS = make(chan *event, 100)

	sio := socketio.NewSocketIO(&config)
	SERVER = NewServerHandler(sio)

	sio.OnConnect(func(c *socketio.Conn) {
		SERVER.OnConnect(c)
		EVENTS <- &event{c, eventConnect, nil}
	})
	sio.OnDisconnect(func(c *socketio.Conn) {
		SERVER.OnDisconnect(c)
		EVENTS <- &event{c, eventDisconnect, nil}
	})
	sio.OnMessage(func(c *socketio.Conn, msg socketio.Message) {
		SERVER.OnMessage(c, msg)
		EVENTS <- &event{c, eventMessage, msg}
	})
	go func() {
		http.ListenAndServe(fmt.Sprintf(SERVER_ADDR), sio.ServeMux())
		EVENTS <- &event{nil, eventCrash, nil}
	}()

	return EVENTS
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

func connectClient(t *testing.T) (*socketio.WebsocketClient, chan *message, chan bool) {
	clientMessage := make(chan *message)
	clientDisconnect := make(chan bool)

	client := socketio.NewWebsocketClient(socketio.SIOCodec{})
	client.OnMessage(func(msg socketio.Message) {
		j, _ := msg.JSON()
		obj, err := SERVER.jsonToData(j)
		if err != nil {
			t.Fatalf("Client received a message that was not valid JSON: %v", j)
		}
		clientMessage <- obj
	})
	client.OnDisconnect(func() {
		clientDisconnect <- true
	})

	err := client.Dial("ws://"+SERVER_ADDR+"/realtime/websocket", "http://"+SERVER_ADDR+"/")
	if err != nil {
		t.Fatal(err)
	}

	// expect connection
	serverEvent := <-EVENTS
	if serverEvent.eventType != eventConnect {
		t.Fatalf("Expected eventConnect but got %v", serverEvent)
	}

	// init
	msg := newInit()
	if err = client.Send(msg); err != nil {
		t.Fatal("Send init:", err)
	}
	serverEvent = <-EVENTS
	if serverEvent.eventType != eventMessage {
		t.Fatalf("Expected eventMessage but got %v", serverEvent)
	}

	// subscribe
	msg = newSub()
	if err = client.Send(msg); err != nil {
		t.Fatal("Send subscribe:", err)
	}
	serverEvent = <-EVENTS
	if serverEvent.eventType != eventMessage {
		t.Fatalf("Expected eventMessage but got %v", serverEvent)
	}
	msg = <-clientMessage
	if msg.Data["count"].(float64) != 1 {
		t.Fatalf("Expected subscription count to be 1 but got %v", msg.Data["count"])
	}

	return client, clientMessage, clientDisconnect
}

func TestMessages(t *testing.T) {
	CONFIG.DEBUG = false

	numMessages := 1000

	serverEvents := startServer()

	time.Sleep(1e9)
	client, clientMessage, clientDisconnect := connectClient(t)

	t.Logf("Sending and receiving %d messages", numMessages)

	iook := make(chan bool)

	go func() {
		var err os.Error
		var m *message
		for i := 0; i < numMessages; i++ {
			m = newMsg()
			m.Data["msg"] = fmt.Sprintf("%d", i)
			if err = client.Send(m); err != nil {
				t.Fatal("Send:", err)
			}
		}
		iook <- true
	}()

	go func() {
		var val string
		for i := 0; i < numMessages; i++ {
			msg := <-clientMessage

			// message data check
			val = fmt.Sprintf("%d", i)
			if msg.Data["msg"] != val {
				t.Fatalf("Expected %v but got %v", val, msg.Data["msg"])
			}

			// identity should always get passed around
			if msg.Identity != IDENT {
				t.Fatalf("Expected idenity to be %v but got %v", IDENT, msg.Identity)
			}
		}
		iook <- true
	}()

	go func() {
		for i := 0; i < numMessages; i++ {
			serverEvent := <-serverEvents
			if serverEvent.eventType != eventMessage {
				t.Fatalf("Expected eventMessage but got %v", serverEvent)
			}
		}
		iook <- true
	}()

	for i := 0; i < 3; i++ {
		<-iook
	}

	// unsubscribe
	msg := newUnsub()
	if err := client.Send(msg); err != nil {
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

	CONFIG.DEBUG = false
}
