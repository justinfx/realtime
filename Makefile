include $(GOROOT)/src/Make.inc

TARG=dist/server
GOFILES=\
	message.go\
	server.go\
	realtime.go\
        
include $(GOROOT)/src/Make.cmd
