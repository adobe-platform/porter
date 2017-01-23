package main

import (
	"fmt"
	"net/http"
	"os"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world\n"))
}

func main() {

	http.HandleFunc("/", Handler)
	http.HandleFunc("/health", Handler)

	port := os.Getenv("PORT")
	fmt.Println("listening on " + port)
	http.ListenAndServe(":"+port, nil)
}
