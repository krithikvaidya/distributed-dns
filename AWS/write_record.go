package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func main() {

	fmt.Printf("Enter key: ")
	var key, value string
	fmt.Scanf("%s", &key)

	fmt.Printf("Enter the number of nameservers: ")
	var n int
	fmt.Scanf("%d", &n)

	var IPs = make([]string, n)

	for i := 0; i < n; i++ {
		fmt.Printf("Enter the IP of %d: ", i+1)
		fmt.Scanf("%s", &IPs[i])
	}

	var operation string
	fmt.Printf("Enter the Operation to be performed: ")
	fmt.Scanf("%s", &operation)

	operation = strings.ToLower(operation)
	if operation != "delete" {
		fmt.Printf("Enter the new value to be stored: ")
		fmt.Scanf("%s", &value)
	}

	success := false
	client := http.Client{}
	for i := 0; i < n; i++ {

		if operation == "update" {

			formData := url.Values{
				"value": {value},
			}

			req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://%s:4000/%s", IPs[i], key), bytes.NewBufferString(formData.Encode()))
			if err != nil {
				log.Printf("\nError in http.NewRequest: %v\n", err)
				break
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("\nError in client.Do: %v\n", err)
				break
			}

			resp.Body.Close()

		} else if operation == "create" {

			formData := url.Values{
				"value": {value},
			}

			url := fmt.Sprintf(fmt.Sprintf("http://%s:4000/%s", IPs[i], key))
			resp, err := http.PostForm(url, formData)

			if err != nil {
				log.Printf("\nError in client.Do: %v\n", err)
				break
			}

			resp.Body.Close()

		} else {
			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://%s:4000/%s", IPs[i], key), nil)
			if err != nil {
				log.Printf("\nError in http.NewRequest: %v\n", err)
				break
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded;")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("\nError in client.Do: %v\n", err)
				continue
			}

			resp.Body.Close()
		}

		if i == n-1 {
			success = true
		}
	}
	if success {
		fmt.Printf("Success\n")
	} else {
		fmt.Printf("Not Successful")
	}
}
