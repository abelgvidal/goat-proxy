package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "I hear you in :9009\n")
	})

	fmt.Println("I hear you in :9009")
	http.ListenAndServe(":9009", nil)
}
