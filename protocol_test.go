package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

type mockConn struct {
	content string
	written chan bool
}

func (conn *mockConn) ReadJSON(v interface{}) error {
	err := json.Unmarshal([]byte(conn.content), v)
	return err
}

func (conn *mockConn) WriteJSON(v interface{}) error {
	bytes, err := json.Marshal(v)
	conn.content = string(bytes)
	conn.written <- true
	return err
}

func TestHubRun(t *testing.T) {
	conn := &mockConn{
		written: make(chan bool),
		content: "",
	}

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
