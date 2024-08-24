package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type Log struct {
	mu      sync.Mutex
	records []Record
}

type Record struct {
	Value  []byte `json:"value"`
	Offset uint64 `json:"offset"`
}

func main() {
	var log Log

	http.HandleFunc("/decode", func(w http.ResponseWriter, r *http.Request) {
		var recordd Record
		log.mu.Lock()
		defer log.mu.Unlock()

		json.NewDecoder(r.Body).Decode(&recordd)
		recordd.Offset = uint64(len(log.records))
		log.records = append(log.records, recordd)
	})

	http.HandleFunc("/encode", func(w http.ResponseWriter, r *http.Request) {
		log.mu.Lock()
		defer log.mu.Unlock()

		json.NewEncoder(w).Encode(log)
	})

	http.ListenAndServe(":8080", nil)
}
