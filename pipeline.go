package main

import (
	"net/http"
	"time"
)

type Pipeline interface {
	Read() chan *http.Request
	Write() chan *http.Request
}

type PipelineChain struct {
	In   chan *http.Request
	Out  chan *http.Request
	Name string
}

type PipelineBlock struct {
	/*
		in --- ---> inner -->f() ---> out
			  |                    ^
			  -------->------------
	*/
	PipelineChain
	inner   chan *http.Request
	F       func(*http.Request) *http.Request
	Started bool
}

func (p PipelineBlock) New(name string, f func(*http.Request) *http.Request) PipelineBlock {
	p.In = make(chan *http.Request, 100)
	p.Out = make(chan *http.Request, 100)
	p.inner = make(chan *http.Request, 100)
	p.Name = name
	p.F = f
	p.Started = false
	p.Start()
	return p
}

func (pc *PipelineChain) Read() chan *http.Request {
	return pc.Out
}

func (pc *PipelineChain) Write() chan *http.Request {
	return pc.In
}

func (pb *PipelineBlock) Start() {
	if !pb.Started {
		pb.Started = true
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
				case req := <-pb.inner:
					filteredReq := pb.F(req)
					select {
					case pb.Out <- filteredReq:
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
				case req := <-pb.In:
					// nil reqs are to be dropped
					if req != nil {
						select {
						case pb.inner <- req:
						case <-time.After(time.Millisecond * 5):
							select {
							case pb.Out <- req:
							default:
								debug("req lost after attempting pipeline bypass")
							}
						}
					}
				}
			}
		}()
	}
}

func connectPipeline(pfirst, psecond PipelineChain) (chain PipelineChain) {

	wg.Add(1)
	go func() {
		defer wg.Done()
	loop:
		for {
			select {
			case <-exiting:
				break loop
				// blocks on read from pfirst
				// drops if write not ready
			case req := <-pfirst.Read():
				select {
				case psecond.Write() <- req:
				default:
					debug("request lost by pipeline connector")
				}
			}
		}
	}()

	return PipelineChain{In: pfirst.Write(), Out: psecond.Read(), Name: pfirst.Name + " | " + psecond.Name}
}
