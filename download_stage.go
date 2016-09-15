package main

import (
	"net/http"
	"strconv"
)

var downloadOutputQ = make(chan *http.Response, bufferSize)

//var httpConfig = make(chan *HTTPConfig)

func httpClient(id int) {
	var doWork = true
	for doWork {
		select {
		case <-exiting:
			debug("Worker ", strconv.Itoa(id), "exiting")
			doWork = false
		case req, more := <-reqFilterInputQ:
			ping()
			if more {
				debug("Worker ", strconv.Itoa(id), ": downloading ", req.URL.String())
				resp, err := client.Do(req)
				if err != nil {
					debug("Worker ", strconv.Itoa(id), ": error downloading ", req.URL.String(), "(", err.Error(), ")")
					continue
				}
				debug("Worker ", strconv.Itoa(id), ": download completed ", req.URL.String())
				ping()
				select {
				case <-exiting:
					debug("Worker ", strconv.Itoa(id), "exiting")
					doWork = false
				case downloadOutputQ <- resp:
					debug("response queued for processing")
				default:
					// DO NOT BLOCK HTTP CLIENTS
					debug("Response was lost (nobody there to pick it)")
				}

			} else {
				debug("Worker ", strconv.Itoa(id), ": request queue is empty and closed")
				doWork = false
			}
			//case config <- httpConfig:
		}
	}

	wg.Done()
	return
}
