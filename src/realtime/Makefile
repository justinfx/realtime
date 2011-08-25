include $(GOROOT)/src/Make.inc

PROJ_ROOT=../../

TARG=$(PROJ_ROOT)realtime
GOFILES=\
	util.go\
	message.go\
	api.go\
	server.go\
	realtime.go\

    
include $(GOROOT)/src/Make.cmd

.PHONY: gofmt, test, export

gofmt:
	gofmt -w *.go

test:
	gotest -test.v

OUT=realtime_export.tar.gz

export: $(TARG)
	-rm $(PROJ_ROOT)log/* 
	-rm $(PROJ_ROOT)run/*
	cd $(PROJ_ROOT) && git-archive-all $(OUT) --extra realtime --extra log/ --extra run/ --prefix RealTime/ -v
