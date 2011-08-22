package main

import (
	"flag"
	"log"
	"http"
	"fmt"
	"strings"
	"os"
	"os/signal"
	"http/pprof"

	// 3rd party
	//"github.com/justinfx/go-socket.io"
	//"github.com/madari/go-socket.io"
	"socketio" // dev only
)

//
// Set up runtime constants
//


var (
	CONFIG  *Config
	ROOT    string
	LICENSE License
)

const (
	CONF_NAME = "realtime.conf"
)

type Config struct {
	DEBUG         bool
	PORT          int
	FLASHPORT     int
	HWM           int
	DOMAINS       []string
	ALLOWED_TYPES []string
}

func init() {

	CONFIG = &Config{
		DEBUG:         false,
		DOMAINS:       []string{"*"},
		ALLOWED_TYPES: []string{},
		PORT:          8001,
		FLASHPORT:     843,
		HWM:           5000,
	}

	var err os.Error
	LICENSE, err = NewLicense()
	if err != nil {
		fmt.Println("Warning: No valid license keys were found. Only localhost connections are permitted.")
	}

}

//
// Run the server
//
func main() {

	// setup and options
	//var domainVal string

	if c, err := getConf(); err == nil {

		if v, e := c.Bool("Server", "debug"); e == nil {
			CONFIG.DEBUG = v
		}
		if v, e := c.Int("Server", "websocket-port"); e == nil {
			CONFIG.PORT = v
		}
		if v, e := c.Int("Messaging", "message-cache-limit"); e == nil {
			CONFIG.HWM = v
		}

		if v, e := c.String("Server", "allowed-types"); e == nil && v != "" {
			types := strings.Split(v, ",")
			for i, s := range types {
				types[i] = strings.TrimSpace(s)
			}
			if len(types) > 0 {
				CONFIG.ALLOWED_TYPES = types
			}
		}
	}

	fDebug := flag.Bool("debug", false, "Print more feedback from the server")
	fPort := flag.Int("port", -1, "Start the server on this port (Default 8001)")

	flag.Parse()

	if *fDebug {
		CONFIG.DEBUG = true
	}
	if *fPort > 0 {
		CONFIG.PORT = *fPort
	}

	log.Printf("Using config options: DEBUG=%v, PORT=%v, HWM=%v",
		CONFIG.DEBUG, CONFIG.PORT, CONFIG.HWM)

	// create the socket.io server
	config := socketio.DefaultConfig
	config.QueueLength = CONFIG.HWM
	config.Origins = CONFIG.DOMAINS
	config.Resource = "/realtime/"

	if len(CONFIG.ALLOWED_TYPES) > 0 {
		config.Transports = make([]socketio.Transport, len(CONFIG.ALLOWED_TYPES))
		for i, t := range CONFIG.ALLOWED_TYPES {
			switch t {
			case "xhr-polling":
				config.Transports[i] = socketio.NewXHRPollingTransport(10e9, 5e9)
			case "xhr-multipart":
				config.Transports[i] = socketio.NewXHRMultipartTransport(0, 5e9)
			case "websocket":
				config.Transports[i] = socketio.NewWebsocketTransport(0, 5e9)
			case "htmlfile":
				config.Transports[i] = socketio.NewHTMLFileTransport(0, 5e9)
			case "flashsocket":
				config.Transports[i] = socketio.NewFlashsocketTransport(0, 5e9)
			case "json-polling":
				config.Transports[i] = socketio.NewJSONPPollingTransport(0, 5e9)
			}
		}
	}

	sio := socketio.NewSocketIO(&config)
	handler := NewServerHandler(sio)

	sio.OnConnect(func(c *socketio.Conn) { handler.OnConnect(c) })
	sio.OnDisconnect(func(c *socketio.Conn) { handler.OnDisconnect(c) })
	sio.OnMessage(func(c *socketio.Conn, msg socketio.Message) { handler.OnMessage(c, msg) })
	sio.SetAuthorization(func(r *http.Request) bool { return LICENSE.CheckHttpRequest(r) })

	// start a signal handler
	go func() {
		for sig := range signal.Incoming {
			switch sig.(os.UnixSignal) {
			case os.SIGTERM, os.SIGINT:
				log.Println("Server shutting down.")
				handler.Shutdown()
				os.Exit(0)
			}
		}
	}()

	// start the flash server
	go func() {
		if err := sio.ListenAndServeFlashPolicy(fmt.Sprintf(":%v", CONFIG.FLASHPORT)); err != nil {
			log.Println("Warning: Could not start flash policy server", err)
		}
	}()

	// mux and server
	mux := sio.ServeMux()
	// this is a temporary static dir for testing
	mux.Handle("/", http.FileServer(http.Dir("www/")))

	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/heap", http.HandlerFunc(pprof.Heap))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))

	// start server
	log.Printf("RealTime server starting. Accepting connections on port :%v", CONFIG.PORT)

	if err := http.ListenAndServe(fmt.Sprintf(":%v", CONFIG.PORT), mux); err != nil {
		log.Fatal("ListenAndServe:", err)
		os.Exit(2)
	}

	os.Exit(0)

}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	return (err == nil)
}

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
