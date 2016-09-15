package main

import (
	"flag"
	"fmt"
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

const banner = `
######## ##          ###    ##    ## ######## ##     ## ########
##       ##         ## ##   ###   ## ##       ##     ## ##     ##
##       ##        ##   ##  ####  ## ##       ##     ## ##     ##
######   ##       ##     ## ## ## ## ######   ##     ## ########
##       ##       ######### ##  #### ##       ##     ## ##   ##
##       ##       ##     ## ##   ### ##       ##     ## ##    ##
##       ######## ##     ## ##    ## ########  #######  ##     ##


`

const workerCount = 20
const bufferSize = 1000

var (
	infoLog          *log.Logger
	debugLog         *log.Logger
	debugMode        bool
	serverMode       bool
	url              string
	activity         chan struct{}
	exiting          chan struct{}
	reqFilterInputQ  chan *http.Request
	reqFilterOutputQ chan *http.Request
	downloadOutputQ  chan *http.Response
	client           *http.Client
	wg               sync.WaitGroup
)

func main() {
	fmt.Println(banner)
	flag.StringVar(&url, "u", "", "Base URL")
	flag.BoolVar(&debugMode, "debug", false, "log additional debug traces")
	flag.BoolVar(&serverMode, "server", false, "launch testing server")

	flag.Parse()

	LogInit(debugMode)

	if serverMode {
		launchServer()
		//initSignals()
		os.Exit(1)
	}

	if url == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Init internal resources and data structures

	activity = make(chan struct{})
	exiting = make(chan struct{})
	timeout := time.Duration(5 * time.Second)
	jar, _ := cookiejar.New(nil)
	client = &http.Client{
		Timeout: timeout,
		Jar:     jar,
	}
	initSignals()
	initWatchdog()

	// Launch Stages

	// Request Filter Stage
	// reqFilterInputQ--->[]--->[]-->. . . . . -->[]--->reqFilterOutputQ

	reqFilterInputQ = make(chan *http.Request, bufferSize)
	reqFilterOutputQ = make(chan *http.Request, bufferSize)
	wg.Add(1)
	go reqFilterPipeline(1)

	// Download Stage
	downloadOutputQ = make(chan *http.Response, bufferSize)
	for id := 1; id <= workerCount; id++ {
		wg.Add(1)
		go httpClient(id)
	}
	// Response Processing Stage
	wg.Add(1)
	go responseProcessor(1)

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
	wg.Wait()
	close(downloadOutputQ)
	close(reqFilterInputQ)
	close(reqFilterOutputQ)

	// Wait for exiting

	info("info test")
	debug("debug test")

}

func broadcastExit(msg string) {
	debug("broadcasting exit from", msg)
	close(exiting)
}

func initSignals() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		select {
		case <-c:
			debug("close exiting because of signal")
			broadcastExit("Interrupt/SIGTERM Signal")
		case <-exiting:
		}
		wg.Done()
		return
	}()
}
func initWatchdog() {
	wg.Add(1)
	go func() {
		doWork := true
		for doWork {
			select {
			case <-activity:
				debug("goroutines: ", strconv.Itoa(runtime.NumGoroutine()))
			case <-exiting:
				debug("watchdog exiting")
				doWork = false
			case <-time.After(time.Second * 10):
				debug("no activity")
				//var once sync.Once
				//once.Do(cleanup)
				debug("close exiting because of inactivity")
				broadcastExit("Idle Timeout")
				doWork = false
			}
		}
		wg.Done()
		return
	}()

}

/*
/////////////////////////
/////////////////////////
UTLIITY FUNCTIONS
/////////////////////////
/////////////////////////

*/

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
