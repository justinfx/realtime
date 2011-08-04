include $(GOROOT)/src/Make.inc

TARG=dist/realtime
GOFILES=\
	util.go\
	message.go\
	server.go\
	realtime.go\
        
include $(GOROOT)/src/Make.cmd

.PHONY: gofmt
gofmt:
	gofmt -w $(GOFILES)
