package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type pipelineStage struct {
	/*
		in --- ---> inner -->f() ---> out
			  |                    ^
			  -------->------------
	*/
	in      chan *http.Request
	out     chan *http.Request
	inner   chan *http.Request
	name    string
	f       func(*http.Request) *http.Request
	Started bool
}

func (p *pipelineStage) init(name string, f func(*http.Request) *http.Request) {
	p.in = make(chan *http.Request, 100)
	p.out = make(chan *http.Request, 100)
	p.inner = make(chan *http.Request, 100)
	p.name = name
	p.f = f
	p.Started = false
}

func (ps *pipelineStage) start() {
	ps.Started = true
	// Inner function
	// Blocks on read from ps.inner pipe
	// Blocks on write to ps.out, timeouts to /dev/null
	// Delays caused by ps.f latency will cause external producers to bypass
	wg.Add(1)
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case <-exiting:
				break loop
			case req := <-ps.inner:
				filteredReq := ps.f(req)
				select {
				case ps.out <- filteredReq:
				case <-time.After(time.Millisecond * 500):
					debug("request lost by f trying to write to outQ")
				}
			}
		}
	}()

	// Inner bypass decider
	// Blocks on read from ps.in
	// timeouts on write to internalPipe (ps.f too slow), writing to ps.out
	// Defaults on write to ps.out
	wg.Add(1)
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case <-exiting:
				break loop
			case req := <-ps.in:
				// nil reqs are to be dropped
				if req != nil {
					select {
					case ps.inner <- req:
					case <-time.After(time.Millisecond * 5):
						select {
						case ps.out <- req:
						default:
							debug("req lost after attempting pipeline bypass")
						}
					}
				}
			}
		}
	}()
}

func connectPipelineStages(pfirst, psecond pipelineStage) (chan *http.Request, chan *http.Request) {

	if !pfirst.Started {
		pfirst.start()
	}
	if !psecond.Started {
		psecond.start()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case <-exiting:
				break loop
			case req := <-pfirst.out:
				select {
				case psecond.in <- req:
				default:
					debug("request lost by pipeline connector")
				}
			}
		}
	}()

	return pfirst.in, psecond.out
}

var dummyStage pipelineStage

func reqFilterPipeline(id int) {
	defer wg.Done()

	dummyStage = pipelineStage{}
	dummyStage.init("dummy", dummyFilter)
	dummyStage.start()

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
