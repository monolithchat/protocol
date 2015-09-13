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
	running          bool
}

type ProcessorFn func(*Hub, *Message) (*Message, error)

func GenericHub() *Hub {
	return &Hub{
		connections:      make(map[Conn]bool),
		globalBroadcasts: make(chan *Message),
		processors:       make(map[string]ProcessorFn),
		running:          false,
	}
}

func (hub *Hub) Run() {
	hub.running = true
	for conn, _ := range hub.connections {
		go hub.listen(conn)
	}

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
	if hub.running {
		go hub.listen(connection)
	}
}

func (hub *Hub) CloseConnection(connection Conn) {
	err := connection.Close()
	if err != nil {
		log.Println("Error closing connection", err)
	} else {
		delete(hub.connections, connection)
	}
}

func (hub *Hub) listen(conn Conn) {
	for {
		request := &Message{}
		err := conn.ReadJSON(request)
		if err != nil {
			log.Println("Error reading JSON")
			hub.CloseConnection(conn)
			log.Println(err)
			return
		}

		fn, exists := hub.processors[request.Type]
		var response *Message
		if exists {
			response, err = fn(hub, request)
			if err != nil {
				response = &Message{
					Type:    "error",
					Payload: err.Error(),
				}
			}
		} else {
			response = &Message{
				Type:    "error",
				Payload: "unknown message type",
			}
		}

		response.stamp()
		err = conn.WriteJSON(response)
		if err != nil {
			hub.CloseConnection(conn)
			log.Println("Closed connection")
			log.Println(err)
			return
		}
	}
}

func (message *Message) stamp() {
	if message.Date.IsZero() {
		message.Date = time.Now()
	}
}
