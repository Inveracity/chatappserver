package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	reql "gopkg.in/gorethink/gorethink.v3"
)

// Handler struct for clients
type Handler func(*Client, interface{})

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Router struct for keeping track of where to send handlers
type Router struct {
	rules   map[string]Handler
	session *reql.Session
}

// NewRouter is the constructor function to initiate the router
func NewRouter(session *reql.Session) *Router {
	return &Router{
		rules:   make(map[string]Handler),
		session: session,
	}
}

// Handle is a struct method to add routes to the rules map
func (r *Router) Handle(msgName string, handler Handler) {
	r.rules[msgName] = handler
}

// FindHandler is a struct method, pass on to the client,
// to figure out what instruction was sent from the frontend
func (r *Router) FindHandler(msgName string) (Handler, bool) {
	handler, found := r.rules[msgName]
	return handler, found
}

// ServeHTTP is a struct method to have the router upgrade the connection to websockets and start serving
// It creates a new client, passes in the socket, handling and database connection and start a go routine for writing data
func (r *Router) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(writer, req, nil)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, err.Error())
		return
	}

	client := NewClient(socket, r.FindHandler, r.session)
	defer client.Close() // prevent leaks by guaranteeing all goroutines are closed
	go client.Write()
	client.Read()

}
