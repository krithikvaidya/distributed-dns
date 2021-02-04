package kv_store

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

func (kv *store) writeFile() {
	dataFile, err := os.Create(kv.filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// serialize the data
	dataEncoder := gob.NewEncoder(dataFile)

	err = dataEncoder.Encode(kv.db)
	log.Printf(err.Error())

	dataFile.Close()
}

func (kv *store) readFile() {

	dataFile, err := os.Open(kv.filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataDecoder := gob.NewDecoder(dataFile)
	dataDecoder.Decode(&kv.db)

	err = dataDecoder.Decode(&kv.db)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataFile.Close()
}

func (kv *store) Recover() {

	kv.mu.RLock()
	defer kv.mu.RUnlock()
	kv.readFile()

}

func (kv *store) Persist() {
	kv.writeFile()
}

func (kv *store) HasData() bool {

	kv.mu.RLock()
	defer kv.mu.RUnlock()

	fi, err := os.Stat(kv.filename)

	if os.IsNotExist(err) {
		return false
	} else if fi.Size() == 0 {
		return false
	}

	return true
}
