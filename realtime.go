package main

import (
	"flag"
	"log"
	"http"
	"fmt"
	"strings"
	"socketio"
	"os"
	"path/filepath"
	"github.com/kless/goconfig/config"
	//	"tideland-rdc.googlecode.com/hg"
	//	"time"
)


//
// Set up runtime constants
//

var (
	CONFIG *Config
	ROOT   string
)

const (
	CONF_NAME = "realtime.conf"
)

type Config struct {
	DEBUG         bool
	DOMAINS       []string
	PORT          int
	FLASHPORT     int
	HWM           int
	ALLOWED_TYPES []string
}

func init() {

	CONFIG = &Config{
		DEBUG:         true,
		DOMAINS:       []string{"*"},
		PORT:          8001,
		FLASHPORT:     843,
		HWM:           15,
		ALLOWED_TYPES: []string{},
	}

	root, _ := filepath.Split(os.Args[0])
	ROOT, _ = filepath.Abs(root)
}

//
// Run the server
//
func main() {

	// setup and options
	var domainVal string

	if c, err := getConf(); err == nil {
		if v, e := c.Bool("Server", "debug"); e == nil {
			CONFIG.DEBUG = v
		}
		if v, e := c.Int("Server", "websocket-port"); e == nil {
			CONFIG.PORT = v
		}
		if v, e := c.String("Server", "allowed-domain"); e == nil {
			domainVal = v
		}
		if v, e := c.Int("Messaging", "message-cache-limit"); e == nil {
			CONFIG.HWM = v
		}

		if v, e := c.String("Server", "allowed-types"); e == nil && v != "" {
			types := strings.Split(v, ",", -1)
			for i, s := range types {
				types[i] = strings.TrimSpace(s)
			}
			if len(types) != 0 {
				CONFIG.ALLOWED_TYPES = types
			}
		}
	}

	fDebug := flag.Bool("debug", false, "Print more feedback from the server")
	fPort := flag.Int("port", -1, "Start the server on this port (Default 8001)")
	fDomains := flag.String("domains", "", "Limit client connections to these comma-sep domain origin:port")
	//flag.IntVar(&(CONFIG.FLASHPORT), "flashport", 843, "Start the flashsocket server on this port (Default 843)")
	flag.Parse()

	if *fDebug {
		CONFIG.DEBUG = true
	}
	if *fPort > 0 {
		CONFIG.PORT = *fPort
	}
	if *fDomains != "" {
		domainVal = *fDomains
	}

	domains := strings.Split(domainVal, ",", -1)
	for i, s := range domains {
		domains[i] = strings.TrimSpace(s)
	}
	if len(domains) != 0 {
		CONFIG.DOMAINS = domains
	}

	Debugf("Using config options: DEBUG=%v, PORT=%v, FLASHPORT=%v, DOMAINS=%v",
		CONFIG.DEBUG, CONFIG.PORT, CONFIG.FLASHPORT, CONFIG.DOMAINS)

	// create the socket.io server
	config := socketio.DefaultConfig
	config.QueueLength = CONFIG.HWM
	config.HeartbeatInterval = 12e9
	config.Resource = "/realtime/"
	config.Origins = CONFIG.DOMAINS

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
	//rd		:= rdc.NewRedisDatabase(rdc.Configuration{})
	handler := NewServerHandler(sio)

	//sio.OnConnect(func(c *socketio.Conn){handler.OnConnect(c)})
	sio.OnDisconnect(func(c *socketio.Conn) { handler.OnDisconnect(c) })
	sio.OnMessage(func(c *socketio.Conn, msg socketio.Message) { handler.OnMessage(c, msg) })

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


func getConf() (*config.Config, os.Error) {
	p1 := filepath.Join(ROOT, CONF_NAME)
	parent, _ := filepath.Split(ROOT)
	p2 := filepath.Join(parent, "etc", CONF_NAME)

	for _, p := range []string{p1, p2} {
		if fileExists(p) {
			if c, err := config.ReadDefault(p); err != nil {
				return nil, os.NewError(fmt.Sprintf("Error reading config: %v", p))
			} else {
				return c, nil
			}
		}
	}

	return nil, os.NewError("No config file found")

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
