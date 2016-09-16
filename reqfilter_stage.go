package main

import (
	"net/http"
	"strconv"
	"time"
)

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
				time.AfterFunc(time.Duration((reqCooldown-(secs-lastTime)))*time.Second, func() {
					defer wg.Done()
					debug("afterfunc")
					select {
					case <-exiting:
					case reqFilterOutputQ <- req:
						ping()
					default:
					}

				})
				originTime[req.URL.Host] = lastTime + reqCooldown

			default:
				continue loop
			}

		}
	}
	return
}
