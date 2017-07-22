package main

import (
	"github.com/gorilla/websocket"
	reql "gopkg.in/gorethink/gorethink.v3"
	"log"
)

// FindHandler is a funcstruct to pass in a handler function to another function
type FindHandler func(string) (Handler, bool)

// Client struct handles websocket and database connections
type Client struct {
	send         chan Message
	socket       *websocket.Conn
	findHandler  FindHandler
	session      *reql.Session
	stopChannels map[int]chan bool
	ID           string
	userName     string
}

// NewStopChannel is a struct method to signal a channel to stop
func (client *Client) NewStopChannel(stopKey int) chan bool {
	client.StopForKey(stopKey)
	stop := make(chan bool)
	client.stopChannels[stopKey] = stop
	return stop
}

// StopForKey helps stopping goroutines when unsubscribing channels
func (client *Client) StopForKey(key int) {
	if ch, found := client.stopChannels[key]; found {
		ch <- true
		delete(client.stopChannels, key)
	}
}

// Read is a struct method to read incoming websocket messages
func (client *Client) Read() {
	var message Message
	for {
		if err := client.socket.ReadJSON(&message); err != nil {
			break
		}

		if handler, found := client.findHandler(message.Name); found {
			handler(client, message.Data)
		}
	}
	client.socket.Close()
}

func (client *Client) Write() {
	for msg := range client.send {
		if err := client.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	client.socket.Close()
}

// Close sends a signal through the signal to close a connection
func (client *Client) Close() {
	for _, ch := range client.stopChannels {
		ch <- true
	}
	close(client.send)

	// Delete user
	reql.Table("user").Get(client.ID).Delete().Exec(client.session)
}

// NewClient does stuff
func NewClient(socket *websocket.Conn, findHandler FindHandler, session *reql.Session) *Client {
	var user User
	user.Name = "anonymous"
	result, err := reql.Table("user").Insert(user).RunWrite(session)
	if err != nil {
		log.Println(err.Error())
	}

	var id string
	if len(result.GeneratedKeys) > 0 {
		id = result.GeneratedKeys[0]
		log.Println("userid: " + id)
	}

	return &Client{
		send:         make(chan Message),
		socket:       socket,
		findHandler:  findHandler,
		session:      session,
		stopChannels: make(map[int]chan bool),
		ID:           id,
		userName:     user.Name,
	}
}
