package main

import (
	"log"
	"time"

	"github.com/mitchellh/mapstructure"
	reql "gopkg.in/gorethink/gorethink.v3"
)

// iota automatically sets the three constants to 0, 1 and 2
const (
	ChannelStop = iota // = 0
	UserStop           // = 1
	MessageStop        // = 2
)

// Message struct is the struct for handling Messages
type Message struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

// Channel struct contains the ID of the channel and the name of the channel
type Channel struct {
	ID   string `json:"id" gorethink:"id,omitempty"`
	Name string `json:"name" gorethink:"name"`
}

// User struct contains the ID of the user and the name of the user
type User struct {
	ID   string `gorethink:"id,omitempty"`
	Name string `gorethink:"name"`
}

// ChannelMessage struct to handle chat messages per channelid
type ChannelMessage struct {
	ID        string    `gorethink:"id,omitempty"`
	ChannelID string    `gorethink:"channelId"`
	Body      string    `gorethink:"body"`
	Author    string    `gorethink:"author"`
	CreatedAt time.Time `gorethink:"createdAt"`
}

// ---------- USER METHODS ----------

// Called when a user changes their name
func editUser(client *Client, data interface{}) {
	var user User
	err := mapstructure.Decode(data, &user)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	client.userName = user.Name
	go func() {
		_, err := reql.Table("user").
			Get(client.ID).
			Update(user).
			RunWrite(client.session)

		if err != nil {
			client.send <- Message{"error", err.Error()}
		}
	}()
}

// Called when a user changes to a new channel
func subscribeUser(client *Client, data interface{}) {
	go func() {
		stop := client.NewStopChannel(UserStop)
		cursor, err := reql.Table("user").
			Changes(reql.ChangesOpts{IncludeInitial: true}).
			Run(client.session)

		if err != nil {
			client.send <- Message{"error", err.Error()}
			return
		}

		changeFeedHelper(cursor, "user", client.send, stop)
	}()
}

// Called when a user changes to a new channel
func unsubscribeUser(client *Client, data interface{}) {
	client.StopForKey(UserStop)
}

// ---------- MESSAGE METHODS ----------

func addChannelMessage(client *Client, data interface{}) {
	var channelMessage ChannelMessage
	err := mapstructure.Decode(data, &channelMessage)

	if err != nil {
		client.send <- Message{"error", err.Error()}
	}

	go func() {
		channelMessage.CreatedAt = time.Now()
		channelMessage.Author = client.userName
		err := reql.Table("message").
			Insert(channelMessage).
			Exec(client.session)

		if err != nil {
			client.send <- Message{"error", err.Error()}
		}

	}()
}

func subscribeChannelMessage(client *Client, data interface{}) {
	go func() {
		eventData := data.(map[string]interface{})
		val, ok := eventData["channelId"]
		if !ok {
			return
		}

		channelID, ok := val.(string)
		if !ok {
			return
		}

		stop := client.NewStopChannel(MessageStop)
		cursor, err := reql.Table("message").
			OrderBy(reql.OrderByOpts{Index: reql.Desc("createdAt")}).
			Filter(reql.Row.Field("channelId").Eq(channelID)).
			Changes(reql.ChangesOpts{IncludeInitial: true}).
			Run(client.session)

		if err != nil {
			client.send <- Message{"error", err.Error()}
			return
		}

		changeFeedHelper(cursor, "message", client.send, stop)

	}()
}

func unsubscribeChannelMessage(client *Client, data interface{}) {
	client.StopForKey(MessageStop)
}

// ---------- CHANNEL METHODS ----------

func addChannel(client *Client, data interface{}) {
	log.Println("adding channel")
	var channel Channel

	err := mapstructure.Decode(data, &channel)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	// Insert a newly created channel into the database, using its own go routine
	go func() {
		err = reql.Table("channel").
			Insert(channel).
			Exec(client.session)

		if err != nil {
			client.send <- Message{"error", err.Error()}
		}
	}()
}

func subscribeChannel(client *Client, data interface{}) {
	go func() {
		stop := client.NewStopChannel(ChannelStop)
		cursor, err := reql.Table("channel").
			Changes(reql.ChangesOpts{IncludeInitial: true}).
			Run(client.session)

		if err != nil {
			client.send <- Message{"error", err.Error()}
			return
		}

		changeFeedHelper(cursor, "channel", client.send, stop)
	}()
}

func unsubscribeChannel(client *Client, data interface{}) {
	client.StopForKey(ChannelStop)
}

// ChangeFeedHelper has three cases, add, remove and edit
// it takes in the change event and sends back the the updated values as they are handled
func changeFeedHelper(cursor *reql.Cursor, changeEventName string, send chan<- Message, stop <-chan bool) {
	log.Println(changeEventName)
	change := make(chan reql.ChangeResponse)
	cursor.Listen(change)

	for {
		eventName := ""
		var data interface{}

		select {
		case val := <-change:
			if val.NewValue != nil && val.OldValue == nil {
				eventName = changeEventName + " add"
				data = val.NewValue
				log.Println(eventName)

			} else if val.NewValue == nil && val.OldValue != nil {
				eventName = changeEventName + " remove"
				data = val.OldValue
				log.Println(eventName)

			} else if val.NewValue != nil && val.OldValue != nil {
				eventName = changeEventName + " edit"
				data = val.NewValue
				log.Println(eventName)
			}

			// Send the event and updated values back to the handler
			send <- Message{eventName, data}

		// If stop is recieved, close the connection to rethinkdb
		case <-stop:
			log.Println("Closing connection")
			cursor.Close()
			return
		}
	}
}
