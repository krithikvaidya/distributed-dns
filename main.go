package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
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

	// Command line parameters
	flag.IntVar(&n_replica, "n", 5, "total number of replicas (default=5)")
	flag.Parse()

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

}

func main() {

	log.Println("Raft-based Replicated Key Value Store")

	log.Printf("Enter the replica's id: ")
	var rid int
	fmt.Scanf("%d", &rid)

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

		rep_addrs[i] = ":500" + strconv.Itoa(i)
	}

	// Perform steps necessary to setup the node as an active replica.
	node.Connect_raft_node(master_context, rid, rep_addrs, false)

	log.Printf("Node initialization successful")

	node.ListenForShutdown(master_cancel)
}
