package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"offchain-oracles/storage"
	"strconv"

	"github.com/syndtr/goleveldb/leveldb"
)

var db *leveldb.DB

func StartServer(host string, newDb *leveldb.DB) {
	for {
		db = newDb
		http.HandleFunc("/api/price/", handleGetPrice)
		http.ListenAndServe(host, nil)
		println("Restart server")
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

	text, err := storage.GetKeystore(db, height)
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
