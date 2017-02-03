package main

import (
	"fmt"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world\n"))
}

func main() {

	http.HandleFunc("/", Handler)
	http.HandleFunc("/health", Handler)

	port := "3000"
	fmt.Println("listening on " + port)
	http.ListenAndServe(":"+port, nil)
}
