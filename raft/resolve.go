package raft

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func resolve() {
	if len(os.Args) < 3 {
		log.Fatalf("Too few arguments, exiting program...")
	}

	dns_name := os.Args[1]
	root_server := os.Args[2]

	fmt.Println(dns_name)
	fmt.Println(root_server)
	keys := strings.Split(dns_name, ".")

	url := fmt.Sprintf("%s:4000/%s", root_server, keys[0])
	//keys[0] = .com / .net /.gov /.edu
	resp, err := http.Get(url)
	if err != nil {
		CheckErrorFatal(err)
	}

	contents, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		CheckErrorFatal(err2)
	}

	text := string(contents)
	resp.Body.Close()

	for i := range keys {
		if i == 0 {
			continue
		}

		arr := strings.Split(text, ",")
		record := arr[0]
		name_server := arr[1]

		if record == "ERR" {
			log.Printf("Failed\n")
			break
		}

		if record == "A" {
			log.Printf("Found : %s", name_server)
			break
		}

		keys[i] = keys[i] + "." + keys[i-1] // example.com
		url = fmt.Sprintf("%s:4000/%s", name_server, keys[1])

		resp, err = http.Get(url)
		if err != nil {
			CheckErrorFatal(err)
		}

		contents, err2 = ioutil.ReadAll(resp.Body)
		if err2 != nil {
			CheckErrorFatal(err2)
		}

		text = string(contents)
		resp.Body.Close()
	}
	//Check last one
	arr := strings.Split(text, ",")
	record := arr[0]
	name_server := arr[1]

	if record == "ERR" {
		log.Printf("Failed\n")
	}

	if record == "A" {
		log.Printf("Found : %s", name_server)
	}
}
