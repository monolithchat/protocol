package protocol

import (
	"fmt"
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
	Close() error
}

type Hub struct {
	connections      map[Conn]bool
	globalBroadcasts chan *Message
	processors       map[string]ProcessorFn
}

type ProcessorFn func(*Hub, *Message) (*Message, error)

type connWrapper struct {
	connection Conn
	closed     chan bool
}

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

func (hub *Hub) RegisterProcessor(messageName string, fn ProcessorFn) error {
	_, exists := hub.processors[messageName]
	if exists {
		return fmt.Errorf("Processor already exists with the name %d", messageName)
	}

	hub.processors[messageName] = fn
	return nil
}

func (hub *Hub) GlobalBroadcast(message *Message) {
	hub.globalBroadcasts <- message
}

func (hub *Hub) Attach(connection Conn) {
	hub.connections[connection] = true
	go hub.listen(&connWrapper{
		connection: connection,
		closed:     make(chan bool),
	})
}

func (hub *Hub) CloseConnection(connection Conn) {
	err := connection.Close()
	if err != nil {
		log.Println("Error closing connection", err)
	} else {
		delete(hub.connections, connection)
	}
}

func (hub *Hub) listen(connection connWrapper) {
	for {
		request := &Message{}
		err := connection.ReadJSON(request)
		if err != nil {
			hub.CloseConnection(connection)
		}
	}
}
