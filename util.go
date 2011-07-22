package main

import "log"

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