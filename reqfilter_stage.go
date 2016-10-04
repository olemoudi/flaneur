package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)



var dummyStage pipelineStage

func reqFilterPipeline(id int) {
	defer wg.Done()

	dummyStage = pipelineStage{}
	dummyStage.init("dummy", dummyFilter)



	pipelineInputQ := dummyStage.in
	pipelineOutQ := dummyStage.out

	debug("pipeline started", strconv.FormatBool(dummyStage.Started))

loop:
	for {
		select {
		case <-exiting:
			debug("ReqFilter Pipeline exiting")
			break loop
		case req := <-reqFilterInputQ:
			if !strings.HasSuffix("."+req.URL.Host, scope) || req.URL.Host == scope {
				continue loop
			}
			_, dup := seen[strings.TrimSpace(req.URL.String())]
			if dup {
				//debug("duped ", req.URL.String())
				continue loop
			}
			seen[strings.TrimSpace(req.URL.String())] = struct{}{}
			select {
			case pipelineInputQ <- req:
				/*
					case <-time.After(time.Millisecond * 5):
						debug("req lost by reqFilterPipeline main loop")
					default:
				*/
			}

		case req := <-pipelineOutQ:
			scheduleRequest(req)
		}
	}
	return
}

func scheduleRequest(req *http.Request) {
	lastTime := originTime[req.URL.Host]
	if lastTime == 0 {
		lastTime = time.Now().Unix()
	}
	now := time.Now()
	secs := now.Unix()

	wg.Add(1)
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case <-time.After(time.Duration((reqCooldown - (secs - lastTime))) * time.Second):
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
