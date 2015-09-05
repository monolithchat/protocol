package protocol

import (
	"log"
	"time"
)

type Message struct {
	Type    string      `json:"type"`
	Date    time.Time   `json:"time"`
	Payload interface{} `json:"payload"`
}

type Conn interface {
	ReadJSON(interface{}) error
	WriteJSON(interface{}) error
}

type Hub struct {
	connections      map[Conn]bool
	globalBroadcasts chan *Message
	processors       map[string]ProcessorFn
}

type ProcessorFn func(*Hub, *Message) (*Message, error)

func GenericHub() *Hub {
	return &Hub{
		connections:      make(map[Conn]bool),
		globalBroadcasts: make(chan *Message),
		processors:       make(map[string]ProcessorFn),
	}
}

func (hub *Hub) Run() {
	for {
		message := <-hub.globalBroadcasts
		for connection := range hub.connections {
			go func(connnection Conn, message *Message) {
				err := connection.WriteJSON(message)
				if err != nil {
					log.Println(err)
				}
			}(connection, message)
		}
	}
}

func (hub *Hub) GlobalBroadcast(message *Message) {
	hub.globalBroadcasts <- message
}
