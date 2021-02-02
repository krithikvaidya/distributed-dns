package kv_store

import (
	"encoding/gob"
	"fmt"
	"os"
)

func (kv *Store) writeFile(filename string) {
	dataFile, err := os.Create(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// serialize the data
	dataEncoder := gob.NewEncoder(dataFile)

	dataEncoder.Encode(kv.db)

	dataFile.Close()
}

func (kv *Store) readFile(filename string) {
	dataFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataDecoder := gob.NewDecoder(dataFile)

	err = dataDecoder.Decode(&kv.db)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataFile.Close()
}

func (kv *Store) Recover(filename string) {

	kv.mu.RLock()
	defer kv.mu.RUnlock()
	kv.readFile(filename)
}

func (kv *Store) Set(filename string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.writeFile(filename)
}

func (kv *Store) HasData(filename string) bool {

	kv.mu.RLock()
	defer kv.mu.RUnlock()

	fi, err := os.Stat(filename)

	if os.IsNotExist(err) {
		return false
	} else if fi.Size() == 0 {
		return false
	}

	return true
}
