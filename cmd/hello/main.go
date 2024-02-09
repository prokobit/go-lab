package main

import (
	"flag"
	"fmt"
	"net/http"
)

func main() {
	port := flag.Int("port", 8080, "Port")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Server listening on http://localhost%s\n", addr)

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		panic(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World")
}
