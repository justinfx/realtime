package main

import (
	"testing"
	//"github.com/justinfx/go-socket.io"
	"socketio"
	"strings"
	"os"
	"log"
	"flag"
	"strconv"
)


func BenchmarkStressTest(b *testing.B) {

	b.ResetTimer()
	b.StopTimer()

	clients := 1
	msg_size := 150 // bytes

	clientDisconnect := make(chan bool)

	numMessages := b.N
	serverAddr := "localhost:8001"

	flag.Parse()
	if len(flag.Args()) > 0 {
		serverAddr = flag.Arg(0)
	}
	c, err := strconv.Atoi(flag.Arg(1))
	if err == nil {
		clients = c
	}
	c, err = strconv.Atoi(flag.Arg(2))
	if err == nil {
		if c > msg_size {
			msg_size = c
		}
	}

	if clients > 1 {
		log.Printf("\nTest starting with %d parallel clients...", clients)
	}

	b.StartTimer()

	for i := 0; i < clients; i++ {
		go func() {
			clientMessage := make(chan socketio.Message)

			log.Println("Connecting to server at:", serverAddr)
			client := socketio.NewWebsocketClient(socketio.SIOCodec{})
			client.OnMessage(func(msg socketio.Message) {
				clientMessage <- msg
			})
			client.OnDisconnect(func() {
				clientDisconnect <- true
			})

			err := client.Dial("ws://"+serverAddr+"/realtime/websocket", "http://"+serverAddr+"/")
			if err != nil {
				log.Fatal(err)
			}

			initCommand := NewCommand()
			initCommand.Data["command"] = "init"

			subCommand := NewCommand()
			subCommand.Channel = "chat"
			subCommand.Data["command"] = "subscribe"

			msgCommand := NewMessage()
			msgCommand.Channel = "chat"
			msgCommand.Data["msg"] = strings.Repeat("X", (msg_size - 53))

			if err = client.Send(initCommand); err != nil {
				log.Fatal("Send init:", err)
			}

			if err = client.Send(subCommand); err != nil {
				log.Fatal("Send init:", err)
			}

			iook := make(chan bool)

			go func() {

				log.Printf("Sending %d messages of size %v bytes...", numMessages, msg_size)
				var err os.Error
				var msg message

				for i := 0; i < numMessages; i++ {
					//time.Sleep(0)
					msg = *msgCommand
					if err = client.Send(msg); err != nil {
						log.Fatal("Send ERROR:", err)
					} else {
						//log.Printf("Sent #%v", i+1)
					}
				}

				iook <- true
			}()

			go func() {
				log.Printf("Receiving messages...")
				for i := 0; i < numMessages; i++ {
					<-clientMessage
				}
				iook <- true
			}()

			for i := 0; i < 2; i++ {
				<-iook
			}

			go func() {
				if err = client.Close(); err != nil {
					log.Fatal("Close ERROR:", err)
				}
			}()
		}()
	}

	log.Println("Waiting for clients disconnect")
	for i := 0; i < clients; i++ {
		<-clientDisconnect
		log.Printf("client #%d finished", i+1)
	}

	log.Printf("Sent %v messages * %v concurrent clients = %v messages", numMessages, clients, numMessages*clients)
}
