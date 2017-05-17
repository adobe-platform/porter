package main

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"
)

func FeatureFlipper(w http.ResponseWriter, r *http.Request) {
	addr := os.Getenv("FEATURE_FLIPPER_ADDR")
	port := os.Getenv("FEATURE_FLIPPER_PORT")
	fmt.Println("FEATURE_FLIPPER_ADDR", addr)
	fmt.Println("FEATURE_FLIPPER_PORT", port)
	resp, err := http.Get("http://" + addr + ":" + port + "/config/features/default")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func EmptyHandler(w http.ResponseWriter, r *http.Request) {
	// do nothing
}

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world\n"))
}

func Health(w http.ResponseWriter, r *http.Request) {
	switch os.Getenv("PORTER_ENVIRONMENT") {
	case "CustomVPC":
		if os.Getenv("CUSTOM_SECRET") == "" {
			w.WriteHeader(500)
		}
	case "CIS":
		if os.Getenv("SOME_S3_SECRET") == "" {
			w.WriteHeader(500)
		}
	}
}

func LoadTest(w http.ResponseWriter, r *http.Request) {
	respTime := r.Header.Get("X-Response-Time")
	if respTime == "" {
		respTime = "1s"
	}

	dur, err := time.ParseDuration(respTime)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	// optionally make outgoing connections
	outgoingURL := r.Header.Get("X-Outgoing-URL")
	if outgoingURL != "" {
		http.Get(outgoingURL)
	}

	time.Sleep(dur)
}

func LeakMemory(w http.ResponseWriter, r *http.Request) {
	leakerLock.Lock()
	defer leakerLock.Unlock()

	leaker[leakerCounter] = garbageStruct()
	leakerCounter++
	w.Write([]byte(fmt.Sprintf("%d", leakerCounter)))
}

func Log(w http.ResponseWriter, r *http.Request) {
	fmt.Println("searchable_porter_log")
}

func Syslog(w http.ResponseWriter, r *http.Request) {
	addr := os.Getenv("RSYSLOG_TCP_ADDR")
	port := os.Getenv("RSYSLOG_TCP_PORT")

	log, err := syslog.Dial("tcp", addr+":"+port, syslog.LOG_INFO|syslog.LOG_DAEMON, "")
	if err != nil {
		fmt.Println("syslog.Dial", err)
		w.WriteHeader(500)
		return
	}

	_, err = fmt.Fprintln(log, "searchable_porter_syslog")
	if err != nil {
		fmt.Println("fmt.Fprintln", err)
		w.WriteHeader(500)
		return
	}
}

func TestLogRotate(w http.ResponseWriter, r *http.Request) {
	for i := 0; i < 100; i++ {
		fmt.Println("fill")
		fmt.Println("this")
		fmt.Println("box")
		fmt.Println("up")
		fmt.Println("with")
		fmt.Println("logs")
	}
}

func Environment(w http.ResponseWriter, r *http.Request) {
	for _, kvp := range os.Environ() {
		w.Write([]byte(kvp + "\n"))
	}
}

var (
	leaker        map[uint64]*Garbage
	leakerCounter uint64
	leakerLock    *sync.Mutex
)

func main() {

	leaker = make(map[uint64]*Garbage)
	leakerLock = new(sync.Mutex)

	for _, kvp := range os.Environ() {
		fmt.Println(kvp)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/empty", EmptyHandler)
	http.HandleFunc("/hello", HelloWorld)
	http.HandleFunc("/custom_health_check", Health)
	http.HandleFunc("/ff", FeatureFlipper)
	http.HandleFunc("/load", LoadTest)
	http.HandleFunc("/leak", LeakMemory)
	http.HandleFunc("/env", Environment)
	http.HandleFunc("/log", Log)
	http.HandleFunc("/syslog", Syslog)
	http.HandleFunc("/logrotate", TestLogRotate)

	fmt.Println("listening on " + port)
	http.ListenAndServe(":"+port, nil)
}

type Garbage struct {
	garbage []byte
}

func garbageStruct() *Garbage {
	return &Garbage{
		garbage: make([]byte, 1024),
	}
}
