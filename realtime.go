package main

import (
	"flag"
	"log"
	"http"
	"fmt"
	"strings"
	"socketio"
//	"tideland-rdc.googlecode.com/hg"
//	"time"
//	"os"
)

type Config struct {
	DEBUG bool
	DOMAINS []string
	PORT int
	FLASHPORT int
	HWM	int
	ALLOWED_TYPES []string
}

var CONFIG	*Config 


func Debugln(v ...interface{}) {
	if CONFIG.DEBUG {
		log.Println(v...)
	}
}

func Debugf(f string, v ...interface{}) {
	if CONFIG.DEBUG {
		log.Printf(f, v...)
	}
}



func main() {
	
	// setup and options
	CONFIG = &Config{
		DEBUG: true,
		DOMAINS: []string{"*"},
		PORT: 8001,
		FLASHPORT: 843,
		HWM: 15,
		ALLOWED_TYPES: []string{"websocket", "flashsocket", "xhr-multipart", "xhr-polling"},
	}
	
	domainVal := ""
	
	flag.BoolVar(&(CONFIG.DEBUG), "debug", false, "Print more feedback from the server")
	flag.IntVar(&(CONFIG.PORT), "port", 8001, "Start the server on this port (Default 8001)")
	flag.IntVar(&(CONFIG.FLASHPORT), "flashport", 843, "Start the flashsocket server on this port (Default 843)")
	flag.StringVar(&domainVal, "domains", "", "Limit client connections to these comma-sep domain origin:port")

	flag.Parse()
	
	domains := strings.Split(domainVal, ",", -1)
	for i, s := range domains {
		domains[i] = strings.TrimSpace(s)
	}	 
	if len(domains) != 0 { CONFIG.DOMAINS = domains }
	
	Debugf("Using config options: DEBUG=%v, PORT=%v, FLASHPORT=%v, DOMAINS=%v", 
		CONFIG.DEBUG, CONFIG.PORT, CONFIG.FLASHPORT, CONFIG.DOMAINS)
		
		
	// create the socket.io server
	config := socketio.DefaultConfig
	config.QueueLength = CONFIG.HWM
	config.HeartbeatInterval = 12e9
	config.Resource = "/realtime/"
	config.Origins = CONFIG.DOMAINS
	
	sio 	:= socketio.NewSocketIO(&config)
	//rd		:= rdc.NewRedisDatabase(rdc.Configuration{})
	handler := NewServerHandler(sio)
	
	
	//sio.OnConnect(func(c *socketio.Conn){handler.OnConnect(c)})
	sio.OnDisconnect(func(c *socketio.Conn){handler.OnDisconnect(c)})
	sio.OnMessage(func(c *socketio.Conn, msg socketio.Message){handler.OnMessage(c, msg)})
	
	
	// start the flash server
	go func() {
		if err := sio.ListenAndServeFlashPolicy(fmt.Sprintf(":%v", CONFIG.FLASHPORT)); err != nil {
			log.Println(err)
		}
	}()
	
	log.Printf("Server starting. Tune your browser to http://localhost:%v/", CONFIG.PORT)


	// mux and server
	mux := sio.ServeMux()
	// this is temporary
	mux.Handle("/", http.FileServer("static/", "/"))

	if err := http.ListenAndServe(fmt.Sprintf(":%v", CONFIG.PORT), mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	
}