package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
)

type server struct {
	Password          string
	ClientConnections map[string]*client
	commands          chan command
	mu                sync.RWMutex
}

func newServer(password string) *server {
	return &server{
		Password:          password,
		ClientConnections: make(map[string]*client),
		commands:          make(chan command),
	}
}

func (s *server) run() {
	// keeps receiving the commands from all clients
	for cmd := range s.commands {
		switch cmd.id {
		case cmdauth:
			s.auth(cmd.client, cmd.args[1], cmd.args[2])
		case cmdpm:
			s.pm(cmd.client, cmd.args)
		case cmdmsg:
			s.msg(cmd.client, cmd.args)
		case cmdquit:
			s.quit(cmd.client)
		}
	}
}

// when ctrl-C is pressed we want to terminate all clients
func (s *server) gracefulexit(cancelFunc context.CancelFunc) {
	signalChan := make(chan os.Signal)

	signal.Notify(signalChan, os.Interrupt)

	for {
		sig := <-signalChan
		switch sig {

		case os.Interrupt:
			log.Printf("SIGINT received, exiting ...")
			cancelFunc()
			return
		}
	}
}

func (s *server) newClient(ctx context.Context, conn net.Conn) {
	c := &client{
		conn:     conn,
		address:  conn.RemoteAddr().String(),
		username: "anonymous",
		commands: s.commands, //initialize to the server's command channel
	}
	//to read commands from the client's telnet shell
	c.readInput(ctx)
}

func (s *server) auth(c *client, check string, name string) {
	//check if it matches with server password
	if s.Password == check {
		//also make sure that every username is unique
		_, ok := s.ClientConnections[name]
		if ok == true {
			c.msg("username already taken, password was correct")
		} else {
			log.Printf("new client %s has joined: %s", name, c.address)
			c.msg("you are now an authenticated user")
			//enter the new client details into the map
			c.username = name
			s.mu.Lock()
			s.ClientConnections[c.username] = c
			s.mu.Unlock()
		}
	} else {
		c.msg("wrong server password, try again")
	}
}

func (s *server) pm(c *client, args []string) {
	msg := strings.Join(args[2:len(args)], " ")
	to := args[1] //receiver name
	s.mu.RLock()
	id := s.ClientConnections[to] //receiver details from the map
	id.msg("private from " + c.username + " : " + msg)
	s.mu.RUnlock()
}

func (s *server) msg(c *client, args []string) {
	msg := strings.Join(args[1:len(args)], " ")
	s.mu.RLock()
	for name, id := range s.ClientConnections {
		if c.username != name { //to every user excpet the person who sent the message
			id.msg("general from " + c.username + " : " + msg)
		}
	}
	s.mu.RUnlock()
}

func (s *server) quit(c *client) {
	log.Printf("client %s has left the chat: %s", c.username, c.address)
	s.mu.Lock()
	delete(s.ClientConnections, c.address) //erase detailts from the map
	s.mu.Unlock()
	c.msg("Good Bye")
	c.conn.Close()
}
