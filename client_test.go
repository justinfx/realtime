package main

import (
	"testing"
	"socketio"
	"json"
	"bytes"
	"strings"
	"os"
	"log"
	"flag"
	"strconv"
)

func messageToBuffer(msg interface{}, buffer *bytes.Buffer) (err os.Error) {
	buffer.Reset()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = json.Compact(buffer, data)
	if err != nil {
		return err
	}
	return
}

func BenchmarkRealtime(b *testing.B) {

	b.ResetTimer()
	b.StopTimer()

	clients := 1
	msg_size := 150 // bytes

	finished := make(chan bool, 1)
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

	initCommand := `{"type":"command","data":{"command":"init"}}`
	subCommand := `{"type":"command","channel":"chat","data":{"command":"subscribe"}}`
	msgCommand := `{"type":"message","channel":"chat","data":{"msg":"` + strings.Repeat("X", (msg_size-53)) + `"}}`

	for i := 0; i < clients; i++ {
		go func() {
			config := socketio.DefaultConfig
			config.Resource = "/realtime/"
			config.QueueLength = numMessages * 2
			config.Codec = socketio.SIOCodec{}
			config.Origins = []string{serverAddr}

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

			if err = client.Send(initCommand); err != nil {
				log.Fatal("Send init:", err)
			}

			if err = client.Send(subCommand); err != nil {
				log.Fatal("Send init:", err)
			}

			/*
				elem := new(bytes.Buffer)

				// Send init message
				msg := NewCommand()
				msg.Data["command"] = "init"
				messageToBuffer(msg, elem)

				if err = client.Send(elem.String()); err != nil {
					log.Fatal("Send init:", err)
				}

				// subscribe
				msg.Channel = "chat"
				msg.Data["command"] = "subsscribe"
				messageToBuffer(msg, elem)

				if err = client.Send(elem.String()); err != nil {
					log.Fatal("Send subscription:", err)
				}
			*/

			iook := make(chan bool)

			go func() {

				log.Printf("Sending %d messages of size %v bytes...", numMessages, len(msgCommand))

				for i := 0; i < numMessages; i++ {
					if err = client.Send(msgCommand); err != nil {
						log.Fatal("Send:", err)
					} else {
						//log.Printf("Client send #%d", i+1)
					}
				}
				/*
					// create the test message
					msg_test := NewMessage()
					msg_test.Channel = "chat"
					msg_test.Data["msg"] = strings.Repeat("X", int(msg_size-149))

					buffer := new(bytes.Buffer)
					messageToBuffer(msg_test, buffer)	

					log.Printf("Sending %d messages of size %v bytes...", numMessages, buffer.Len())

					for i := 0; i < numMessages; i++ {
						if err = client.Send(buffer.String()); err != nil {
							log.Fatal("Send:", err)
						} else {
							//log.Printf("Client send #%d", i+1)
						}
					}
				*/

				iook <- true
			}()

			go func() {
				log.Printf("Receiving messages...")
				for i := 0; i < numMessages; i++ {
					<-clientMessage
					b.SetBytes(int64(msg_size))
				}
				iook <- true
			}()

			for i := 0; i < 2; i++ {
				<-iook
			}

			go func() {
				if err = client.Close(); err != nil {
					log.Fatal("Close:", err)
				}
			}()
		}()
	}

	log.Println("Waiting for client disconnect")
	for i := 0; i < clients; i++ {
		log.Printf("client #%d finished", i+1)
		<-clientDisconnect
	}

	finished <- true
	log.Printf("Sent %v messages * %v concurrent clients = %v messages", numMessages, clients, numMessages*clients)
}
