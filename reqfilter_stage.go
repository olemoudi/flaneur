package main

import "net/http"

func reqFilterPipeline(id int) {
	loop:
		for  {
			select {
			case <-exiting:
				debug("ReqFilter Pipeline exiting")
				break loop
			default:
				var req *http.Request

				select {
				case req = <- reqFilterInputQ:
				default:
					continue loop
				}
				select {
				case reqFilterOutputQ <- req:
				default:
				continue loop
			}
				ping()
			}
		}

	wg.Done()
	return
}
