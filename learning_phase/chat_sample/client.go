package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

//various commands numbered from 0, 1, 2, 3...
const (
	cmdauth = iota
	cmdpm
	cmdmsg
	cmdquit
)

type command struct {
	id     int
	client *client
	args   []string
}

type client struct {
	conn     net.Conn
	address  string
	username string
	commands chan<- command //incoming channel
}

func (c *client) readInput(ctx context.Context) {
	for {
		select {
		case <-ctx.Done(): //when cancel is called
			log.Printf("Terminating Clients")
			os.Exit(0)
		default:
			msg, err := bufio.NewReader(c.conn).ReadString('\n')
			if err != nil {
				return
			}
			//parse the string
			msg = strings.Trim(msg, "\r\n")

			args := strings.Split(msg, " ")
			cmd := strings.TrimSpace(args[0])
			//send respective command details to server channel
			switch cmd {
			case "/auth":
				c.commands <- command{
					id:     cmdauth,
					client: c,
					args:   args,
				}
			case "/pm":
				c.commands <- command{
					id:     cmdpm,
					client: c,
					args:   args,
				}
			case "/msg":
				c.commands <- command{
					id:     cmdmsg,
					client: c,
					args:   args,
				}
			case "/quit":
				c.commands <- command{
					id:     cmdquit,
					client: c,
				}
			default:
				c.err(fmt.Errorf("unknown command: %s", cmd))
			}
		}
	}
}

func (c *client) err(err error) {
	c.conn.Write([]byte("err: " + err.Error() + "\n"))
}

func (c *client) msg(msg string) {
	c.conn.Write([]byte("> " + msg + "\n"))
}
