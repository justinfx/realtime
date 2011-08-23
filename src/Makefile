include $(GOROOT)/src/Make.inc

TARG=../realtime
GOFILES=\
	util.go\
	message.go\
	server.go\
	realtime.go\

    
include $(GOROOT)/src/Make.cmd

.PHONY: gofmt, test, export
gofmt:
	gofmt -w $(GOFILES)

test:
	gotest -test.v

OUT=realtime_export.tar.gz

export: $(TARG)
	-rm ../log/* 
	-rm ../run/*
	cd .. && git-archive-all $(OUT) --extra realtime --extra log/ --extra run/ -v
