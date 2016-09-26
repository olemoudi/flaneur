package main

import "strconv"

//var httpConfig = make(chan *HTTPConfig)

func httpClient(id int) {
	defer wg.Done()
loop:
	for {
		select {
		case <-exiting:
			debug("Worker ", strconv.Itoa(id), "exiting")
			break loop
		case req, more := <-reqFilterOutputQ:
			ping()
			if more {
				//debug("Worker ", strconv.Itoa(id), ": downloading ", req.URL.String())
				resp, err := client.Do(req)
				if err != nil {
					debug("Worker ", strconv.Itoa(id), ": error downloading ", req.URL.String(), "(", err.Error(), ")")
					continue
				}
				//originTimeMutex.Lock()
				//originTime[req.URL.Host] = time.Now().Unix()
				//originTimeMutex.Unlock()
				debug("Worker ", strconv.Itoa(id), ": download completed ", req.URL.String())
				ping()
				if resp.ContentLength > 10000000 {
					debug("file too big")
					continue
				}
				select {
				case <-exiting:
					debug("Worker ", strconv.Itoa(id), "exiting")
					break loop
				case downloadOutputQ <- resp:
					debug("response queued for processing")
					ping()
				default:
					// DO NOT BLOCK HTTP CLIENTS
					debug("Response was lost (nobody there to pick it)")
				}

			} else {
				debug("Worker ", strconv.Itoa(id), ": request queue is empty and closed")
				break loop
			}
			//case config <- httpConfig:
		}
	}
	return
}
