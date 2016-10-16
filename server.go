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
	Tests map[string]Test
	S     string
}

type Test struct {
	Path      string
	Desc      string
	Name      string
	Validator func(*http.Request) bool
}

var Tests map[string]Test

func launchServer() {
	//	makeTests()
	info("launching server at :8000")
	// global handler = polite and dupe tests
	r := mux.NewRouter()
	r.HandleFunc("/", globalHandler(rootHandler))
	r.HandleFunc("/tests/{TestID}/{TestPath}", globalHandler(TestHandler))

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)
}

//func makeTests()

/*
vars := mux.Vars(request)
category := vars["category"]
*/

func TestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t, prs := ts[vars["TestID"]]
	debug(t.Name)
	if !prs {
		http.Redirect(w, r, fmt.Sprintf("/?error=%s", vars["TestID"]), 301)
	}
	if !t.Validator(r) {
		testFailed(r, t)
	}
	debug("hola")
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf("html/%s.html", vars["TestID"])))
	tmpl.Execute(w, t)

}

func globalHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		fail, diff := politenessTest(r)
		if fail {
			fmt.Println("Politeness Test failed by", strconv.Itoa(int(diff)))
		}
		if dupe(r) {
			fmt.Println("URL visited twice:", r.URL.String())
		}
		fn(w, r)
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	tmpl := template.Must(template.ParseFiles("html/index.html"))
	m := getTests()
	context := Context{Tests: m}
	tmpl.Execute(w, context)
}

func getTests() map[string]Test {
	ts := make(map[string]Test)
	t := Test{
		Name: "DupeTest",
		Desc: "dupe test",
		Path: "/tests/DupeTest",
		Validator: func(r *http.Request) bool {
			url := r.URL.String()
			if url == "/tests/DupeTest" {
				return false
			}
			if strings.HasSuffix(url, "?") {
				return false
			}
			if strings.HasSuffix(url, "#") {
				return false
			}
			if strings.HasSuffix(url, "%20") {
				return false
			}
			if strings.HasSuffix(url, "/") {
				return false
			}
			return true
		},
	}
	ts[t.Name] = t

	t = Test{
		Name:      "DupeTest2",
		Desc:      "dupe test2",
		Path:      "/tests/DupeTest2",
		Validator: func(r *http.Request) bool { return true },
	}
	ts[t.Name] = t

	return ts
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

func dupe(r *http.Request) bool {
	u := strings.TrimSpace(r.URL.String())
	_, dup := seen[u]
	if dup {
		return true
	}
	seen[u] = struct{}{}
	return false
}

func testFailed(r *http.Request, t Test) {
	info(fmt.Sprintf("[%s] TEST FAILED - %s", t.Name, r.URL.String()))
}
