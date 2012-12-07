# Realtime Server
### A socket.io based message server written in Go

Realtime is a message server allowing http web clients to communicate with eachother over a simple interface. 
It consists of both the server, and the client API, both wrapping around socket.io

The package is also wrapped up to integrate with [Supervisor](https://github.com/Supervisor/supervisor) for managing the process.

Currently this version of the server only support socket.io 0.6.x  
There is apparently a newer fork of [go-socket.io compatible to 0.9.0](http://code.google.com/p/go-socketio/), 
so maybe someone will update RealTime to that version, once it has determined to be stable.

Detailed client information and examples can be found here:
http://connectai.com/realtime


---------------------

## Features

  * Simple Javascript client API for connecting and communicating
  * "Channels" support for different communication groups/rooms
  * "Identity" for a user to group multiple connection (browser windows, etc) into a single identity on the server
  * Monitor URL - Can specific a URL endpoint that will receive POST requests notifying when various events occur in the message server
  * Public messages via POST requests
  * Events - Builtin server events like "onSubscribe/onUnsubscribe" and arbitrary client-side events.

## Installation

**Binary builds** (if available):  
https://github.com/justinfx/realtime/downloads

**From source**:

To get the entire application with all support files:

```
git clone --recursive git://github.com/justinfx/realtime.git
cd realtime
./src/build.sh
```

RealTime should now be built into the application directory, and can be directly started:  
`./realtime -port=8001`

You can also use the packages `Supervisor` process manager with the bundled commands:

```
./start
./status
./stop
./restart
```

If you want the flash socket server support to work, then RealTime should be started with sudo, as flash needs a privileged port to run:  
`sudo ./realtime`

## Configuration

Settings can be specified in the `etc/` directory.

  * realtime.conf - Settings specific to the RealTime server process
  * supervisord-realtime.conf - Settings to control how Supervisor will run and manage the RealTime process 
  * supervisord.conf - The Supervisor-specific conf

**License checking**

By default, the server will only accept connections from web clients originating on the localhost. 
License checking is done by comparing the clients request sha1("domain.com"+SECRET). The SECRET is curently stored in the binary (`util.go`) but should probably be moved to the config to be loaded at runtime.

Example:

To allow clients from "mydomain.com" to connect to the RealTime server, generate a sha1 key and add it to the `etc/license.txt`  

```
echo -n mydomain.comRk8ohYJQBXopu82XmVTFsAgG3r4f | shasum -a 1 | awk '{print $1}'
571ab3357c3e56e20b764f25e62149229f5d4b08
```
