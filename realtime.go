package main

import (
	"flag"
	"log"
	"socketio"
	"http"
	"fmt"
	"strings"
//	"time"
//	"os"
//	"redis"
)

type Config struct {
	DEBUG bool
	DOMAINS []string
	PORT int
	FLASHPORT int
	HWM	int
	ALLOWED_TYPES []string
}


var (
	SERVER 	*socketio.SocketIO
	CONFIG	*Config 
)

func main() {
	
	CONFIG = &Config{
		DEBUG: true,
		DOMAINS: nil,
		PORT: 8001,
		FLASHPORT: 843,
		HWM: 15,
		ALLOWED_TYPES: []string{"websocket", "flashsocket", "xhr-multipart", "xhr-polling"},
	}
	
	domainVal := ""
	
	flag.BoolVar(&(CONFIG.DEBUG), "debug", false, "Print more feedback from the server")
	flag.IntVar(&(CONFIG.PORT), "port", 8001, "Start the server on this port (Default 8001)")
	flag.IntVar(&(CONFIG.FLASHPORT), "flashport", 843, "Start the flashsocket server on this port (Default 843)")
	flag.StringVar(&domainVal, "domain", "", "Limit client connections to this domain origin")

	flag.Parse()
	
	domains := strings.Split(domainVal, ",", -1)
	for i, s := range domains {
		domains[i] = strings.TrimSpace(s)
	}	 
	if len(domains)==0 { CONFIG.DOMAINS = domains }
	
	Debugf("Using config options: DEBUG=%v, PORT=%v, FLASHPORT=%v, DOMAINS=%v", 
		CONFIG.DEBUG, CONFIG.PORT, CONFIG.FLASHPORT, CONFIG.DOMAINS)
		
	// create the socket.io server
	config := socketio.DefaultConfig
	config.QueueLength = 100000
	config.HeartbeatInterval = 12e9
	config.Resource = "/realtime/"
	config.Origins = []string{fmt.Sprintf("localhost:%v", CONFIG.PORT)}
	SERVER := socketio.NewSocketIO(&config)
	
	handler := ServerHandler{SERVER}
	
	SERVER.OnConnect(func(c *socketio.Conn){handler.OnConnect(c)})
	SERVER.OnDisconnect(func(c *socketio.Conn){handler.OnDisconnect(c)})
	SERVER.OnMessage(func(c *socketio.Conn, msg socketio.Message){handler.OnMessage(c, msg)})
	
	// start the flash server
	go func() {
		if err := SERVER.ListenAndServeFlashPolicy(fmt.Sprintf(":%v", CONFIG.FLASHPORT)); err != nil {
			log.Println(err)
		}
	}()
	
	log.Printf("Server starting. Tune your browser to http://localhost:%v/", CONFIG.PORT)

	// mux and server
	mux := SERVER.ServeMux()
	mux.Handle("/", http.FileServer("static/", "/"))

	if err := http.ListenAndServe(fmt.Sprintf(":%v", CONFIG.PORT), mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	
}