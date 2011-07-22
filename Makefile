include $(GOROOT)/src/Make.inc

TARG=dist/server
GOFILES=\
	util.go\
	message.go\
	server.go\
	realtime.go\
        
include $(GOROOT)/src/Make.cmd
