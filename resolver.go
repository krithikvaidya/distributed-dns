package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Too few arguments, exiting program...")
	}

	dns_name := os.Args[1]
	root_server := os.Args[2]

	keys := strings.Split(dns_name, ".")

	// reverse keys array
	for i := len(keys)/2 - 1; i >= 0; i-- {
		opp := len(keys) - 1 - i
		keys[i], keys[opp] = keys[opp], keys[i]
	}

	fmt.Printf("\nResolving: %v\n", keys)

	url := fmt.Sprintf("http://%s:4000/%s", root_server, keys[0])

	// keys[0] = com / net / gov / edu
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("%v", err)
	}

	contents, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		log.Fatalf("%v", err)
	}

	text := string(contents)
	resp.Body.Close()

	for i := range keys {

		if i == 0 {
			continue
		}

		arr2 := strings.Split(text, " ")

		if strings.TrimSpace(arr2[len(arr2)-1]) == "pair" {
			log.Fatalf("\nRecord not found\n")
		}

		arr := strings.Split(arr2[len(arr2)-1], ",")

		record := arr[0]
		name_server := strings.TrimSpace(arr[1])

		fmt.Printf("\nResolver: Found %v record: %v\n", record, name_server)

		if record == "A" {
			fmt.Printf("Found : %s", name_server)
			break
		}

		keys[i] = keys[i] + "." + keys[i-1] // example.com
		url = fmt.Sprintf("http://%s:4000/%s", name_server, keys[i])

		resp, err = http.Get(url)
		if err != nil {
			log.Fatalf("%v", err)
		}

		contents, err2 = ioutil.ReadAll(resp.Body)
		if err2 != nil {
			log.Fatalf("%v", err)
		}

		text = string(contents)
		resp.Body.Close()
	}

	arr2 := strings.Split(text, " ")
	arr := strings.Split(arr2[len(arr2)-1], ",")

	if strings.TrimSpace(arr2[len(arr2)-1]) == "pair" {
		log.Fatalf("\nRecord not found\n")
	}

	record := arr[0]
	name_server := strings.TrimSpace(arr[1])

	fmt.Printf("\nResolver: Found %v record: %v\n", record, name_server)

	if record == "A" {
		fmt.Printf("\nFound A record: %s\n", name_server)
	}

}
