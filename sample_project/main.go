package main

import (
	"fmt"
	"net/http"
)

const port = "3000"

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world\n"))
}

func main() {

	http.HandleFunc("/", Handler)
	http.HandleFunc("/health", Handler)

	fmt.Println("listening on " + port)
	http.ListenAndServe(":"+port, nil)
}
