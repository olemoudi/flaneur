package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/willf/bloom"
	//"fmt"
	"log"
	"os"
	"time"
	//"github.com/op/go-logging"
	"net/http"
	"net/http/cookiejar"
	"os/signal"
	"syscall"
)

const workerCount = 25
const bufferSize = 1000
const reqCooldown = 1
const BLOOMSIZE = 20000000 // bits
const BLOOMHASHES = 4      // no of hashes

var (
	infoLog          *log.Logger
	debugLog         *log.Logger
	debugMode        bool
	serverMode       bool
	starturl         string
	scope            string
	activity         chan struct{}
	exiting          chan struct{}
	reqFilterInputQ  chan *http.Request
	reqFilterOutputQ chan *http.Request
	downloadOutputQ  chan *http.Response
	client           *http.Client
	wg               sync.WaitGroup
	originTime       map[string]int64
	originTimeMutex  *sync.Mutex
	bfilter          *bloom.BloomFilter
)

func main() {
	fmt.Println(banner)
	flag.StringVar(&starturl, "u", "", "Base URL")
	flag.BoolVar(&debugMode, "debug", false, "log additional debug traces")
	flag.BoolVar(&serverMode, "server", false, "launch testing server")

	flag.Parse()

	LogInit(debugMode)

	if serverMode {
		launchServer()
		//initSignals()
		os.Exit(1)
	}

	if starturl == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Init internal resources and data structures

	activity = make(chan struct{})
	originTimeMutex = &sync.Mutex{}
	exiting = make(chan struct{})
	timeout := time.Duration(5 * time.Second)
	jar, _ := cookiejar.New(nil)

	//TODO: add redirect policy
	client = &http.Client{
		Timeout: timeout,
		Jar:     jar,
		// CheckRedirect specifies the policy for handling redirects.
		// If CheckRedirect is not nil, the client calls it before
		// following an HTTP redirect. The arguments req and via are
		// the upcoming request and the requests made already, oldest
		// first. If CheckRedirect returns an error, the Client's Get
		// method returns both the previous Response (with its Body
		// closed) and CheckRedirect's error (wrapped in a url.Error)
		// instead of issuing the Request req.
		// As a special case, if CheckRedirect returns ErrUseLastResponse,
		// then the most recent response is returned with its body
		// unclosed, along with a nil error.
		//
		// If CheckRedirect is nil, the Client uses its default policy,
		// which is to stop after 10 consecutive requests.
		//CheckRedirect func(req *Request, via []*Request) error
	}
	originTime = make(map[string]int64)
	bfilter = bloom.New(BLOOMSIZE, BLOOMHASHES)
	debug("Bloom filter estimated FP Rate after 1M URLs:", strconv.FormatFloat(bfilter.EstimateFalsePositiveRate(1000000), 'f', -1, 64))
	<-time.After(time.Second * 4)

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
	for i := 1; i <= extractorCount; i++ {
		wg.Add(1)
		go linkExtractor()
	}
	//wg.Add(1)
	//go responseProcessor(1)

	// out queue

	// launch response broker
	//go responseBroker()
	// out queue
	//processQ = make(chan *http.Response, 1000)

	// launch response processors

	// seed start URL
	req, _ := http.NewRequest("GET", starturl, nil)
	tokens := strings.Split(req.URL.Host, ".")
	scope = strings.Join(tokens[len(tokens)-2:], ".")
	<-time.After(time.Second)
	for i := 1; i <= workerCount; i++ {
		req, _ := http.NewRequest("GET", starturl, nil)
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

//TODO: wg.Wait() timeout
/*
// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
    c := make(chan struct{})
    go func() {
        defer close(c)
        wg.Wait()
    }()
    select {
    case <-c:
        return false // completed normally
    case <-time.After(timeout):
        return true // timed out
    }
}
*/
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
	loop:
		for {
			select {
			case <-activity:
				debug("goroutines: ", strconv.Itoa(runtime.NumGoroutine()))
			case <-exiting:
				debug("watchdog exiting")
				break loop
			case <-time.After(time.Second * 300):
				debug("no activity")
				//var once sync.Once
				//once.Do(cleanup)
				debug("close exiting because of inactivity")
				broadcastExit("Idle Timeout")
				break loop
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
	logfile, err := os.OpenFile("/tmp/vito.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error opening log file")
	}
	infowriter := io.MultiWriter(logfile, os.Stdout)

	if debug_flag {
		debuglogfile, err := os.OpenFile("/tmp/vito.debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
