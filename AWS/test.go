package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
)

func main() {

	cmd := exec.Command("wget", "-q", "-O", "-", "http://169.254.169.254/latest/meta-data/instance-id")
	inst_id, err := cmd.Output()

	cmd = exec.Command("ec2metadata", "--availability-zone", "|", "sed 's/.$//'")
	op, _ := cmd.Output()
	region := strings.Trim(string(op), " ")
	region = strings.Trim(string(op), "\n")
	region2 := region[:len(region)-1]

	fmt.Println(region2)

	req_data := map[string]string{
		"region":  region2,
		"inst_id": string(inst_id),
	}

	jsonValue, _ := json.Marshal(req_data)

	resp, err := http.Post("http://ec2-54-236-50-24.compute-1.amazonaws.com:5000/register_replica", "application/json", bytes.NewBuffer(jsonValue))
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v", err)
	}
	bodyString := string(bodyBytes)
	fmt.Printf(bodyString)
}
