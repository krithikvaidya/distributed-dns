package main

/**

Protocol for sending/receiving messages:
- Each message will be of length 256 bytes.
- There will be 6 types of messages:

1. authenticate~<server_password>~<client_username>~\n  --> sent from client to server
2. authenticated~\n  --> sent from server to client
3. pm~<username>~<message>~\n  --> from client to server, unicast message
4. broadcast~<message>~\n  --> from client to server, broadcast message
5. message~<username>~<type>~<message>~\n  --> from server to client(s). Type indicates unicast/broadcast.
6. terminate~<reason>~\n  --> from server to client, for graceful shutdown

*/

import (
	
)

var (
	// Define variables for command line parameters here if using init() function
)



func main() {

	/**
	- Create a cancellable context to be passed to client/server functions
	- Listen for appropriate termination signals (refer to https://gobyexample.com/signals) in a separate goroutine
	- The above goroutine should call the context's cancel function on receiving a termination signal,
	  and wait for the client/server function called below to complete its execution (hint: using channels)
	- The main function should then decide whether the execution will be in client mode or in server mode.
	  On deciding, it should create & initialize a new struct of client/server type (which will be defined 
	  in client.go and server.go), and call the respective Run() function  
	*/

}

