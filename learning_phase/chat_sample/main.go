package main

import (
	"context"
	"fmt"
	"log"
	"net"
)

var (
	password string
)

func main() {
	fmt.Println("Enter a password for the server")
	fmt.Scanln(&password)

	//cancel if server is shutdown
	ctx, cancelFunc := context.WithCancel(context.Background())

	s := newServer(password) // create instance of new server
	go s.run()               // run server using goroutine

	go s.gracefulexit(cancelFunc) // if ctrl-C is used

	listener, err := net.Listen("tcp", "localhost"+":"+"8888")
	if err != nil {
		log.Fatalf("unable to start server: %s", err.Error())
	}

	defer listener.Close() // keep listening for new connections till the end
	log.Printf("server started on :8888")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %s", err.Error())
			continue
		}
		// run every client separately as a goroutine
		go s.newClient(ctx, conn)
	}
}
