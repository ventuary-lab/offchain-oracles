package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"offchain-oracles/storage"
	"strconv"
)

const (
	defaultHost = "127.0.0.1:8080"
)

func main() {
	var host string
	flag.StringVar(&host, "host", defaultHost, "set host")
	flag.Parse()
	http.HandleFunc("/api/price/", handler)
	http.ListenAndServe(host, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["height"]

	if !ok || len(keys[0]) < 1 {
		http.Error(w,"Url Param 'height' is missing", 404)
		return
	}

	height, err := strconv.Atoi(keys[0])
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	text, err := storage.GetKeystore(height)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	result, err := json.Marshal(text)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Fprintf(w, string(result))
}
