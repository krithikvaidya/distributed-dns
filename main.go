package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/krithikvaidya/distributed-dns/raft"
)

var n_replica int

func init() {

	/*
	 * Workaround for a Go bug
	 * The Init() function for the testing package should be called
	 * before our init() function for parsing the command-line arguments
	 * of the `go test` command
	 */
	testing.Init()

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

	f, err := os.OpenFile("node_logs", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

}

func main() {

	log.Println("Raft-based Replicated Key Value Store")

	raft.RegisterWithOracle()

	raft.GetEnvFromOracle()

	n_replica, _ := strconv.Atoi(os.Getenv("N_REPLICA"))

	// Get replica id from env variable
	rid, _ := strconv.Atoi(os.Getenv("REPLICA_ID"))
	log.Printf("Replica ID = %v\n", rid)

	master_context, master_cancel := context.WithCancel(context.Background())

	node := raft.Setup_raft_node(master_context, rid, n_replica, false)

	node.Meta.Master_ctx = master_context
	node.Meta.Master_cancel = master_cancel

	// Store the gRPC address of other replicas
	rep_addrs := make([]string, n_replica)
	for i := 0; i < n_replica; i++ {
		if i == rid {
			continue
		}

		rep_i_addr := "REP_" + strconv.Itoa(i) + "_INTERNAL_IP"
		rep_addrs[i] = os.Getenv(rep_i_addr) + ":5000"
		log.Printf("\nrep_addrs[i]: %v\n", rep_addrs[i])
	}

	// Perform steps necessary to setup the node as an active replica.
	node.Connect_raft_node(master_context, rid, rep_addrs, false)

	log.Printf("Node initialization successful")

	node.ListenForShutdown(master_cancel)
}
