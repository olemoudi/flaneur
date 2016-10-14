package main

import (
	"net/http"
	"runtime"
	"time"
)

var pipeline Pipeline

func reqFilterPipeline(id int) {
	defer wg.Done()

	dummyBlock := NewPipeline("dummy", dummyFilter)
	normalizeBlock := NewPipeline("normalize", normalizeURL)
	urlseenBlock := NewPipeline("urlseen", urlSeen)

	pipeline = connectPipeline(dummyBlock, normalizeBlock)
	pipeline = connectPipeline(pipeline, urlseenBlock)

loop:
	for {
		select {
		case <-exiting:
			debug("ReqFilter Pipeline exiting")
			break loop
		case req := <-reqFilterInputQ:
			ping()
			pipeline.Write() <- req
		case req := <-pipeline.Read():
			ping()
			if req != nil {
				if runtime.NumGoroutine() < 50000 {
					scheduleRequest(req)
				}
			}
		}
	}
	return
}

func scheduleRequest(req *http.Request) {
	lastTime := originTime[req.URL.Host]
	if lastTime == 0 {
		lastTime = time.Now().Unix() - reqCooldown
	}
	now := time.Now()
	secs := now.Unix()

	delay := reqCooldown - (secs - lastTime)
	wg.Add(1)
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case <-time.After(time.Duration(delay) * time.Second):
				select {
				case reqFilterOutputQ <- req:
					ping()
					break loop
				default:
					debug("req lost by scheduled goroutine")
					break loop
				}
			case <-exiting:
				//debug("cancelling request at", strconv.Itoa(int((reqCooldown - (secs - lastTime)))))
				break loop
			}

		}
	}()
	originTime[req.URL.Host] = lastTime + reqCooldown
}
