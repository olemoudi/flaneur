package main

import "net/http"

var reqFilterInputQ = make(chan *http.Request, bufferSize)
var reqFilterOutputQ = make(chan *http.Request, bufferSize)

func reqFilterPipeline(id int) {
	var doWork = true
	for doWork {
		select {
		case <-exiting:
			debug("ReqFilter Pipeline exiting")
			doWork = false
		default:
			req := <-reqFilterInputQ
			reqFilterOutputQ <- req
			ping()
		}
	}

	wg.Done()
	return
}
