package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/kv_store"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

var n_replica int

// Start the local key-value store and the HTTP server it listens for requests on.
func (node *RaftNode) StartKVStore(ctx context.Context, addr string, num int) {

	filename := "600" + strconv.Itoa(num)

	// InitializeStore is defined in kv_store/restaccess_key_value.go
	kv := kv_store.InitializeStore(filename)

	r := mux.NewRouter()

	r.HandleFunc("/kvstore", kv.KvstoreHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", kv.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", kv.DeleteHandler).Methods("DELETE")

	// Create a server struct
	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	node.meta.kv_store_server = srv

	// Gracefully shut down the server if context is cancelled
	go func() {

		// Block till context is cancelled
		<-ctx.Done()

		// Shut down the server. On error, forcefully close the server
		// We use HTTP's inbuilt Shutdown() and Close() methods for this.
		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

		if err_kv := node.meta.kv_store_server.Shutdown(ctx); err_kv != nil {

			log.Printf("HTTP server Shutdown error: %v\n", err_kv)
			node.meta.kv_store_server.Close()

		}

	}()

	err := node.meta.kv_store_server.ListenAndServe()

	// Handling code for when the server is unexpectedly closed.
	if (err != nil) && (err != http.ErrServerClosed) {
		CheckErrorFatal(err)
	}

	node.meta.shutdown_chan <- "KV store shutdown successful."

}

/*
 * External function to shut down the kv store server.
 */
func (node *RaftNode) StopKVStore() {

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	if err := node.meta.kv_store_server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown error: %v\n", err)
		node.meta.kv_store_server.Close()
	}

}

// Start HTTP server to listen for client requests
func (node *RaftNode) StartRaftServer(ctx context.Context, addr string) {

	node.meta.nodeAddress = addr // store address of the node

	r := mux.NewRouter()

	r.HandleFunc("/test", node.TestHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", node.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", node.DeleteHandler).Methods("DELETE")

	// Create a server struct
	raft_server := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	node.meta.raft_server = raft_server

	// Gracefully shut down the server if context is cancelled
	go func() {

		// Block till context is cancelled
		<-ctx.Done()

		// Shut down the server. On error, forcefully close the server
		// We use HTTP's inbuilt Shutdown() and Close() methods for this.
		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

		if err_kv := node.meta.raft_server.Shutdown(ctx); err_kv != nil {
			log.Printf("HTTP server Shutdown error: %v\n", err_kv)
			node.meta.raft_server.Close()
		}

	}()

	err := node.meta.raft_server.ListenAndServe()

	// Handling code for when the server is unexpectedly closed.
	if (err != nil) && (err != http.ErrServerClosed) {
		CheckErrorFatal(err)
	}

	node.meta.shutdown_chan <- "Client request listener shutdown successful."
}

/*
 * External function to gracefully shut down the raft server
 */
func (node *RaftNode) StopRaftServer() {

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	// Stop the raft store service
	if err := node.meta.raft_server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown error: %v\n", err)
		node.meta.kv_store_server.Close()
	}

}

/*
 * This function starts the gRPC server for the raft node and shuts it down when
 * context is cancelled.
 */
func (node *RaftNode) StartGRPCServer(ctx context.Context, grpc_address string, listener *net.TCPListener) {

	// Shut down the gRPC server if the context is cancelled
	go func() {

		// Block till the context is cancelled
		<-ctx.Done()

		// Stop the server
		node.meta.grpc_server.Stop()

		node.meta.shutdown_chan <- "gRPC server shutdown successful."
	}()

	// Start the server
	log.Printf("\nStarting gRPC server at address %v...\n", grpc_address)
	err := node.meta.grpc_server.Serve(listener)

	CheckErrorFatal(err)
}

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

	f, err := os.OpenFile("logs", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

}

/*
 * This function initializes the node and imports the persistent
 * state information to the node.
 */
func setup_raft_node(ctx context.Context, id int, n_replicas int, testing bool) *RaftNode {

	// Key value store address of the current node
	kv_addr := ":300" + strconv.Itoa(id)

	// InitializeNode is defined in raft_node.go
	node := InitializeNode(int32(n_replicas), id, kv_addr)

	// ApplyToStateMachine() is defined in raft_node.go
	go node.ApplyToStateMachine(ctx)

	// Starting KV store
	kvstore_addr := ":300" + strconv.Itoa(id)
	log.Println("Starting local key-value store...")
	go node.StartKVStore(ctx, kvstore_addr, id)

	/*
	 * Make a HTTP request to the test endpoint until a reply is obtained, indicating that
	 * the HTTP server is up
	 */
	test_addr := fmt.Sprintf("http://localhost%s/kvstore", kvstore_addr)

	if !testing {
		for {

			_, err := http.Get(test_addr)

			if err == nil {
				log.Printf("\nKey-value store up and listening at port %s\n", kvstore_addr)
				break
			}

		}
	}

	return node
}

/*
 * This function connects an existing node to a raft system.
 *
 * It connects the current node to the other nodes. This mechanism includes the
 * initiation of their various services, like the
 * KV store server, the gRPC server and the Raft server.
 *
 * The `connect_chan` channel is used to signify the end of execution of this
 * function for synchronization and error handling.
 */
func (node *RaftNode) connect_raft_node(ctx context.Context, id int, rep_addrs []string, testing bool) {

	/*
	 * Connect the new node to the existing nodes.
	 * Attempt to gRPC dial to other replicas and obtain corresponding client stubs.
	 * ConnectToPeerReplicas is defined in raft_node.go.
	 */
	log.Println("Obtaining client stubs of gRPC servers running at peer replicas...")
	node.ConnectToPeerReplicas(ctx, rep_addrs)

	// Setting up and running the gRPC server
	grpc_address := ":500" + strconv.Itoa(id)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", grpc_address)
	CheckErrorFatal(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	CheckErrorFatal(err)

	node.meta.grpc_server = grpc.NewServer()

	/*
	 * ConsensusService is defined in protos/replica.proto
	 * RegisterConsensusServiceServer is present in the generated .pb.go file
	 */
	protos.RegisterConsensusServiceServer(node.meta.grpc_server, node)

	// Running the gRPC server
	go node.StartGRPCServer(ctx, grpc_address, listener)

	// TODO: wait till grpc server up

	// Now we can start listening to client requests

	// Set up the server that listens for client requests.
	server_address := ":400" + strconv.Itoa(id)
	log.Println("Starting raft replica server...")
	go node.StartRaftServer(ctx, server_address)

	test_addr := fmt.Sprintf("http://localhost%s/test", server_address)

	// Check whether the server is active
	if !testing {
		for {

			_, err = http.Get(test_addr)

			if err == nil {
				log.Printf("\nRaft replica server up and listening at port %s\n", server_address)
				break
			}

		}
	}

}

func main() {

	log.Println("Raft-based Replicated Key Value Store")

	// Get replica id from env variable
	rid, _ := strconv.Atoi(os.Getenv("REPLICA_ID"))
	log.Printf("Replica ID = %v\n", rid)

	master_context, master_cancel := context.WithCancel(context.Background())

	node := setup_raft_node(master_context, rid, n_replica, false)

	node.meta.master_ctx = master_context
	node.meta.master_cancel = master_cancel

	// Store the gRPC address of other replicas
	rep_addrs := make([]string, n_replica)
	for i := 0; i < n_replica; i++ {
		if i == rid {
			continue
		}

		rep_i_addr := "REP_" + strconv.Itoa(i) + "_GRPC_ADDR"
		rep_addrs[i] = os.Getenv(rep_i_addr) + ":500" + strconv.Itoa(i)
	}

	// Perform steps necessary to setup the node as an active replica.
	node.connect_raft_node(master_context, rid, rep_addrs, false)

	log.Printf("Node initialization successful")

	node.ListenForShutdown(master_cancel)
}
