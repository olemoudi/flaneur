package main

import (
	"flag"
	"io"
	"io/ioutil"
	"sync"
	//"fmt"
	"log"
	"os"
	"time"
	//"github.com/op/go-logging"
	"net/http"
	"net/http/cookiejar"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

const workerCount = 20
const processorCount = 1
const bufferSize = 1000

var (
	infoLog          *log.Logger
	debugLog         *log.Logger
	debugMode        bool
	url              string
	links            chan string
	allDone          chan bool
	activity         chan struct{}
	exiting          chan struct{}
	watchdog         chan struct{}
	client           *http.Client
	wg               sync.WaitGroup
	activityWatchdog bool
)

func ping() {
	select {
	case activity <- struct{}{}:
	default:
	}
}

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

func cleanup() {
	debug("cleaning up")
	close(exiting)
	wg.Wait()
	close(downloadOutputQ)
	close(reqFilterInputQ)
	close(reqFilterOutputQ)
	allDone <- true
}

func main() {
	flag.StringVar(&url, "u", "", "Base URL")
	flag.BoolVar(&debugMode, "debug", false, "log additional debug traces")

	flag.Parse()

	if url == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Init internal resources and data structures
	LogInit(debugMode)
	activity = make(chan struct{})
	allDone = make(chan bool)
	exiting = make(chan struct{})
	watchdog = make(chan struct{})

	timeout := time.Duration(5 * time.Second)
	jar, _ := cookiejar.New(nil)
	client = &http.Client{
		Timeout: timeout,
		Jar:     jar,
	}

	initSignals()

	// Launch Stages

	// Request Filter Stage
	wg.Add(1)
	go reqFilterPipeline(1)

	// Download Stage
	for id := 1; id <= workerCount; id++ {
		wg.Add(1)
		go httpClient(id)
	}

	// Response Processing Stage

	for id := 1; id <= processorCount; id++ {
		wg.Add(1)
		go responseProcessor(id)
	}

	// out queue

	// launch response broker
	//go responseBroker()
	// out queue
	//processQ = make(chan *http.Response, 1000)

	// launch response processors

	// seed start URL
	for i := 1; i <= workerCount; i++ {
		req, _ := http.NewRequest("GET", url, nil)
		reqFilterInputQ <- req
	}

	// sync workers
	<-allDone

	// Wait for exiting

	info("info test")
	debug("debug test")

}

func initSignals() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		debug("signal")
		cleanup()
	}()

	go func() {
		doWork := true
		for doWork {
			select {
			case <-activity:
				debug("goroutines: ", strconv.Itoa(runtime.NumGoroutine()))
			case <-exiting:
				debug("watchdog exiting")
				doWork = false
			case <-time.After(time.Second * 30):
				debug("no activity")
				//var once sync.Once
				//once.Do(cleanup)
				cleanup()
			}
		}
		return
	}()

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
