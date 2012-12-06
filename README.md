# Realtime Server
### A socket.io based message server written in Go

Realtime is a message server allowing http web clients to communicate with eachother over a simple interface. 
It consists of both the server, and the client API, both wrapping around socket.io

The package is also wrapped up to integrate with [Supervisor](https://github.com/Supervisor/supervisor) for managing the process.

Currently this version of the server only support socket.io 0.6.x  
There is apparently a newer fork of [go-socket.io compatible to 0.9.0](http://code.google.com/p/go-socketio/)

The client code is included as a submodule and directly location here:
https://github.com/justinfx/realtime-client

Detailed client information and examples can be found here:
http://connectai.com/realtime

---------------------

The server by itself can be installed directly with go install:  
`go install github.com/justinfx/realtime/src/realtime`

But to get all the support tools and package structure, you should clone the entire repository.