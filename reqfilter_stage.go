package main

import (
	"net/http"
	"strconv"
	"time"
)

type pipelineStage struct {
	in     chan *http.Request
	inBypass chan *http.Request
	out    chan *http.Request
	outBypass	chan *http.Request
	name   string
	f      func(*http.Request) chan *http.Request
}

func createPipelineStage(stageName string, fu func(*http.Request) chan *http.Request) pipelineStage {
	inChan := make(chan *http.Request)
	inChanBypass := make(chan *http.Request)
	outChan := make(chan *http.Request)
	outChanBypass := make(chan *http.Request)
	stage = pipelineStage { in : inChan, bypass : bypassChan, out: outChan, name : stageName, f: fu}
	wg.Add(1)
	go func() {
		defer wg.Done()
		loop:
			for {
				select {
				case <-exiting:
					break loop
				case req <-bypass:
					select {
						case outReq
					}
				case req<- stage.in:
					select {
					case outReq <- stage.f(req):
					case <-time.After(Time.Seconds*1):

					}
				}
			}

	}
	return pipelineStage{}
}

func (a *pipelineStage) connectPipeline(b *pipelineStage) pipelineStage {

}

func (p *pipelineStage) send(req *http.Request) {

	select {
	case p.in <- req:
	default:
		debug(p.name, "bypassed")
		p.out <- req
	}

}

func reqFilterPipeline(id int) {
	defer wg.Done()

loop:
	for {
		select {
		case <-exiting:
			debug("ReqFilter Pipeline exiting")
			break loop
		default:
			var req *http.Request

			select {
			case req = <-reqFilterInputQ:
				lastTime := originTime[req.URL.Host]
				if lastTime == 0 {
					lastTime = time.Now().Unix()
				}
				now := time.Now()
				secs := now.Unix()
				debug("Next in:", strconv.Itoa(int((reqCooldown - (secs - lastTime)))))

				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case <-time.After(time.Duration((reqCooldown - (secs - lastTime))) * time.Second):
						select {
						case reqFilterOutputQ <- req:
							ping()
						default:
						}
					case <-exiting:
						//debug("cancelling request at", strconv.Itoa(int((reqCooldown - (secs - lastTime)))))
					}
				}()

				originTime[req.URL.Host] = lastTime + reqCooldown

			default:
				continue loop
			}

		}
	}
	return
}
