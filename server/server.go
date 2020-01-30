package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"offchain-oracles/storage"
	"strconv"
)

var dbPath string

func StartServer(host string, newDbPath string) {
	for {
		dbPath = newDbPath
		http.HandleFunc("/api/price/", handleGetPrice)
		http.ListenAndServe(host, nil)
	}
}

func handleGetPrice(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["height"]

	if !ok || len(keys[0]) < 1 {
		http.Error(w, "Url Param 'height' is missing", 404)
		return
	}

	height, err := strconv.Atoi(keys[0])
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	text, err := storage.GetKeystore(dbPath, height)
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
