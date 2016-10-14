package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

type Context struct {
	Path string
}

type Test interface {
	Name() string
	Desc() string
	Path() string
	Validate(*http.Request) bool
}

type TestInstance struct {
	Path string
	Name string

	Validator func(http.ResponseWriter, *http.Request)
}

var Tests map[string]Test

func launchServer() {
	makeTests()
	info("launching server at :8000")
	// global handler = polite and dupe tests
	http.HandleFunc("/", globalHandler(rootHandler))

	r := mux.NewRouter()
	r.HandleFunc("/", globalHandler(rootHandler))
	r.HandleFunc("/tests/{TestID}", globalHandler(TestHandler))

	http.ListenAndServe(":8000", nil)
}

func makeTests()

func TestHandler(w http.ResponseWriter, req *http.Request) {

}

func globalHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pt, diff := politenessTest(r)
		if pt {
			fmt.Println("Politeness Test failed by", strconv.Itoa(int(diff)))
		}
		if dupeTest(r) {
			fmt.Println("URL visited twice:", r.URL.String())
		}
		fn(w, r)
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	tmpl := template.Must(template.ParseFiles("html/index.html"))

	context := Context{"/"}
	tmpl.Execute(w, context)

}

func testHandler(w http.ResponseWriter, req *http.Request) {

}

func getTests() []Test {
	tests := make([]Test, 0)
	t := Test{
		Name:      "DupeTest",
		Path:      "/tests/DupeTest",
		Handler:   dupeTestHandler,
		Validator: nil,
	}
	tests = append(tests, t)

	return tests
}

/*
func makeTestHandler(fn func(req *http.Request) (bool, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Here we will extract the page title from the Request,
		// and call the provided handler 'fn'
	}
}
*/

var timestamp time.Time = time.Time{}
var politeness int64 = 5
var mutex = &sync.Mutex{}

func politenessTest(req *http.Request) (bool, int64) {
	mutex.Lock()
	defer mutex.Unlock()
	result := false
	niltime := time.Time{}
	var diff int64 = 5
	if timestamp == niltime {
		timestamp = time.Now()
	} else {
		diff = time.Now().Unix() - timestamp.Unix()
		if diff < politeness {
			result = true
			timestamp = time.Now()
		} else {
			result = false
		}
	}

	return result, diff
}

var seen map[string]interface{} = make(map[string]interface{})

func dupeTest(r *http.Request) bool {
	u := strings.TrimSpace(r.URL.String())
	_, dup := seen[u]
	if dup {
		return true
	}
	seen[u] = struct{}{}
	return false
}

const rootTemplate = `
<!DOCTYPE html>
<html>
<head lan="en">
    <meta charset="UTF-8">
    <title>Root Template</title>
</head>
<body>
    <a href="{{.URL}}">link</a>
</body>
</html>
`
