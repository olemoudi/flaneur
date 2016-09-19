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
