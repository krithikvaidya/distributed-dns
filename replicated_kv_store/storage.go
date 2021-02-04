package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
)

type Storage struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

func NewStorage() *Storage {

	gob.Register([]protos.LogEntry{})

	m := make(map[string]interface{})
	return &Storage{
		m: m,
	}
}

func (stored *Storage) WriteFile(filename string) {
	dataFile, err := os.Create(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// serialize the data
	dataEncoder := gob.NewEncoder(dataFile)

	err = dataEncoder.Encode(stored.m)

	if err != nil {
		log.Printf("Error in WriteFile: %v", err.Error())
	}

	dataFile.Close()
}

func (stored *Storage) ReadFile(filename string) {
	dataFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataDecoder := gob.NewDecoder(dataFile)

	err = dataDecoder.Decode(&stored.m)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataFile.Close()
}

func (stored *Storage) Get(key string, filename string) (interface{}, bool) {

	stored.mu.RLock()
	defer stored.mu.RUnlock()
	stored.ReadFile(filename)
	value, check := stored.m[key]
	return value, check
}

func (stored *Storage) Set(key string, value interface{}) {
	stored.mu.Lock()
	defer stored.mu.Unlock()
	stored.m[key] = value
}

func (stored *Storage) HasData(filename string) bool {

	stored.mu.RLock()
	defer stored.mu.RUnlock()

	fi, err := os.Stat(filename)

	if os.IsNotExist(err) {
		return false
	} else if fi.Size() == 0 {
		return false
	}

	return true
}
