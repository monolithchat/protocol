package protocol

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type mockConn struct {
	content string
	written chan bool
	closed  bool
}

func (conn *mockConn) ReadJSON(v interface{}) error {
	if conn.closed {
		return errors.New("Connection is closed")
	}

	err := json.Unmarshal([]byte(conn.content), v)
	return err
}

func (conn *mockConn) WriteJSON(v interface{}) error {
	if conn.closed {
		return errors.New("Connection is closed")
	}

	bytes, err := json.Marshal(v)
	conn.content = string(bytes)
	conn.written <- true
	return err
}

func (conn *mockConn) Close() error {
	close(conn.written)
	conn.closed = true
	return nil
}

func newMockConn() *mockConn {
	return &mockConn{
		written: make(chan bool),
		content: "",
		closed:  true,
	}
}

func TestHubRun(t *testing.T) {
	conn := newMockConn()
	hub := GenericHub()
	go hub.Run()
	hub.connections[conn] = true

	message := Message{
		Type:    "test",
		Date:    time.Now(),
		Payload: "testing!",
	}

	hub.GlobalBroadcast(&message)

	<-conn.written

	var readMessage Message
	err := conn.ReadJSON(&readMessage)
	if err != nil {
		t.Error(err)
	}

	if readMessage.Type != message.Type {
		t.Error("messages not of same type", readMessage.Type)
	}
	if readMessage.Date != message.Date {
		t.Error("messages don't have same date.", readMessage.Date)
	}
	if readMessage.Payload != message.Payload {
		t.Error("messages don't have same payload.", readMessage.Payload)
	}
}

func TestHubAttach(t *testing.T) {
	hub := GenericHub()
	conn := newMockConn()

	hub.Attach(conn)

	_, exists := hub.connections[conn]
	if !exists {
		t.Error("Hub didn't add connection")
	}
}

func TestHubConnectionClosed(t *testing.T) {
	hub := GenericHub()
	conn := newMockConn()
	hub.Attach(conn)

	hub.CloseConnection(conn)
	if !conn.closed {
		t.Error("connection didn't close")
	}
	if hub.connections[conn] {
		t.Error("connection still in hub's map")
	}
}

func TestHubListen(t *testing.T) {

}
