package main

import (
	"flag"
	"io"
	"io/ioutil"
	//"fmt"
	"log"
	"os"
	"time"
	//"github.com/op/go-logging"
	"net/http"
	"net/http/cookiejar"
	"os/signal"
	"strconv"
	"syscall"
)

const workerCount = 20
const processorCount = 0
const bufferSize = 1000

var (
	infoLog   *log.Logger
	debugLog  *log.Logger
	debugMode bool
	url       string
	reqQ      chan *http.Request
	respQ     chan *http.Response
	processQ  chan *http.Response
	allDone   chan bool
	activity  chan struct{}
	finish    chan struct{}
	client    *http.Client
)

func info(msg ...string) {
	s := make([]interface{}, len(msg))
	for i, v := range msg {
		s[i] = v
	}
	infoLog.Println(s...)
}

func debug(msg ...string) {
	s := make([]interface{}, len(msg))
	for i, v := range msg {
		s[i] = v
	}
	debugLog.Println(s...)
}

func LogInit(debug_flag bool) {
	logfile, err := os.OpenFile("vito.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error opening log file")
	}
	infowriter := io.MultiWriter(logfile, os.Stdout)

	if debug_flag {
		debuglogfile, err := os.OpenFile("vito.debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("Error opening debug log file")
		}

		infowriter = io.MultiWriter(logfile, os.Stdout, debuglogfile)

		debugwriter := io.MultiWriter(debuglogfile, os.Stdout)
		debugLog = log.New(debugwriter, "[DEBUG] ", log.Ldate|log.Ltime)

	} else {
		debugLog = log.New(ioutil.Discard, "", 0)
	}

	infoLog = log.New(infowriter, "", log.Ldate|log.Ltime)

}

func responseBroker() {
	for {
		select {
		case finish <- struct{}{}:
			debug("broker broke")
			return
		case resp := <-respQ:
			activity <- struct{}{}
			// process resp
			debug("Response processed")
			for i := 1; i <= 2; i++ {
				req, _ := http.NewRequest("GET", resp.Request.URL.String(), nil)
				select {
				case reqQ <- req:
				case <-time.After(time.Millisecond * 30):
					debug("Request lost")
				}
			}
		}
	}
}

func httpClient(id int) {
	for {
		select {
		case finish <- struct{}{}:
			debug("client broke")
			return
		case req, more := <-reqQ:
			activity <- struct{}{}
			if more {
				debug("Worker ", strconv.Itoa(id), ": downloading ", req.URL.String())
				resp, err := client.Do(req)
				if err != nil {
					debug("Worker ", strconv.Itoa(id), ": error downloading ", req.URL.String())
					continue
				}
				debug("Worker ", strconv.Itoa(id), ": download completed ", req.URL.String())
				respQ <- resp
			} else {
				debug("Worker ", strconv.Itoa(id), ": reqQ is empty and closed")
				return
			}
		}
	}
}

var cleanup = func() {
	debug("cleaning up")
	for {
		select {
		case <-finish:
			continue
		case <-time.After(time.Second * 10):
			close(reqQ)
			close(respQ)
			allDone <- true
			return
		}
	}
}

func responseProcessor(id int) {
}
func main() {
	flag.StringVar(&url, "u", "", "Base URL")
	flag.BoolVar(&debugMode, "debug", false, "log additional debug traces")

	flag.Parse()

	if url == "" {
		flag.Usage()
		os.Exit(1)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Init internal resources and data structures
	LogInit(debugMode)
	activity = make(chan struct{})
	allDone = make(chan bool)
	finish = make(chan struct{}, 1)
	finish <- struct{}{}
	timeout := time.Duration(5 * time.Second)
	jar, _ := cookiejar.New(nil)
	client = &http.Client{
		Timeout: timeout,
		Jar:     jar,
	}

	go func() {
		<-c
		debug("signal")
		cleanup()
	}()

	// Launch pool of http clients
	// in queue
	reqQ = make(chan *http.Request, bufferSize)
	for id := 1; id <= workerCount; id++ {
		go httpClient(id)
	}
	// out queue
	respQ = make(chan *http.Response, bufferSize)

	// launch response broker
	go responseBroker()
	// out queue
	processQ = make(chan *http.Response, 100)

	// launch response processors
	for id := 1; id <= processorCount; id++ {
		go responseProcessor(id)
	}

	// seed start URL
	for i := 1; i <= workerCount; i++ {
		req, _ := http.NewRequest("GET", url, nil)
		reqQ <- req
	}

	// launch done watchdog
	go func() {
		for {
			select {
			case finish <- struct{}{}:
				return
			case <-activity:
			case <-time.After(time.Second * 30):
				debug("no activity")
				//var once sync.Once
				//once.Do(cleanup)
				cleanup()
				return
			}
		}
	}()

	// sync workers
	<-allDone

	// Wait for finish

	info("info test")
	debug("debug test")

}
