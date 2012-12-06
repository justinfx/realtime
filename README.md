# Realtime Server
### A socket.io based message server written in Go

Realtime is a message server allowing http web clients to communicate with eachother over a simple interface. 
It consists of both the server, and the client API, both wrapping around socket.io

The package is also wrapped up to integrate with Supervisor for managing the process.

Currently this version of the server only support socket.io 0.6.x
There is apparently a newer fork of go-socket.io compatible to 0.9.0 here:
http://code.google.com/p/go-socketio/
