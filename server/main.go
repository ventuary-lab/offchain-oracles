package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"offchain-oracles/storage"
	"strconv"
)

func main() {
	http.HandleFunc("/api/price/", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {

	keys, ok := r.URL.Query()["height"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'height' is missing")
		return
	}

	height, err := strconv.Atoi(keys[0])
	if err != nil {
		log.Print(err)
		return
	}

	text, err := storage.GetKeystore(height)
	if err != nil {
		log.Print(err)
		return
	}

	result, err := json.Marshal(text)
	if err != nil {
		log.Print(err)
		return
	}

	fmt.Fprintf(w, string(result))

}
