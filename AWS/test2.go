package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GetEnvResponse struct {
	Message      string   `json:message`
	N_replicas   int      `json:n_replicas`
	Replica_id   int      `json:replica_id`
	Internal_ips []string `json:internal_ips`
	Tg_arn       string   `json:tg_arn`
	Instance_ids []string `json:instance_ids`
}

func main() {

	req_data := map[string]string{
		"region":  "us-east-1",
		"inst_id": "i-08869900d12c4219e",
	}

	jsonValue, _ := json.Marshal(req_data)

	resp, err := http.Post("http://ec2-54-236-50-24.compute-1.amazonaws.com:5000/get_env", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("%v", err)
	}
	defer resp.Body.Close()

	val := &GetEnvResponse{}
	// data, err := ioutil.ReadAll(resp.Body)
	// errj := json.Unmarshal(data, val)

	// if errj != nil {
	// 	fmt.Printf("\nErr: %v\n", errj)
	// }

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v", err)
	}
	bodyString := string(bodyBytes)
	fmt.Printf(bodyString)

	// errj := json.Unmarshal(data, val)
	// var sec map[string]interface{}
	if err = json.Unmarshal(bodyBytes, val); err != nil {

	}
	fmt.Printf("\n%T\n", val.Instance_ids)

	// val := &GetEnvResponse{}
	// decoder := json.NewDecoder(resp.Body)
	// err = decoder.Decode(val)

	// if err != nil {
	// 	fmt.Printf("\nErr: %v\n", err)
	// }
	// fmt.Printf("\n%v\n", val.message)

	// if val.message == "Waiting for more instances to register" {
	// 	fmt.Printf("nice.")
	// }

}
