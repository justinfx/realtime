include $(GOROOT)/src/Make.inc

TARG=dist/realtime
GOFILES=\
	util.go\
	message.go\
	server.go\
	realtime.go\

GOTESTFILES=\
	realtime_test.go\
     
include $(GOROOT)/src/Make.cmd

.PHONY: gofmt, test
gofmt:
	gofmt -w $(GOFILES)

test:
	gotest -test.v