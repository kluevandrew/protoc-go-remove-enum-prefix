package main

import (
	"log"
)

var verbose = false

func logf(format string, arguments ...interface{}) {
	if !verbose {
		return
	}

	log.Printf(format, arguments...)
}
