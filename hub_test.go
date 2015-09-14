package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
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

	<-conn.written
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
	if conn.closed {
		return nil
	}

	//close(conn.written)
	conn.closed = true
	return nil
}

func newMockConn() *mockConn {
	return &mockConn{
		written: make(chan bool),
		content: "",
		closed:  false,
	}
}

func testSetup() (*Hub, Conn) {
	return GenericHub(), newMockConn()
}

func TestHub(t *testing.T) {
	Convey("Hub", t, func() {
		Convey(".RegisterProcessor", func() {
			hub := GenericHub()
			fn := func(hub *Hub, request *Message) (*Message, error) {
				return request, nil
			}

			So(hub.processors, ShouldBeEmpty)
			hub.RegisterProcessor("test", fn)
			So(hub.processors, ShouldNotBeEmpty)
			So(hub.processors["test"], ShouldEqual, ProcessorFn(fn))
		})

		Convey(".Attach", func() {
			hub, conn := testSetup()

			hub.Attach(conn)

			So(hub.connections, ShouldContainKey, conn)
		})

		Convey(".Run", func() {
			hub, conn := testSetup()
			hub.Attach(conn)
			hub.RegisterProcessor("test", func(hub *Hub, request *Message) (*Message, error) {
				return request, nil
			})

			go hub.Run()

			message := Message{
				Type:    "test",
				Date:    time.Now(),
				Payload: "testing!",
			}

			hub.GlobalBroadcast(&message)

			var readMessage Message
			So(conn.ReadJSON(&readMessage), ShouldBeNil)
			So(readMessage, ShouldResemble, message)
		})

		Convey(".ConnectionClose", func() {
			hub, conn := testSetup()
			hub.Attach(conn)

			hub.CloseConnection(conn)

			So(conn.(*mockConn).closed, ShouldBeTrue)
			So(hub.connections, ShouldNotContainKey, conn)
		})

		Convey(".listen", func() {
			Convey("responds with an error when it doesn't know the message type", func() {
				hub, conn := testSetup()

				go hub.listen(conn)

				conn.WriteJSON(&Message{
					Type: "notamethod",
				})

				var response Message
				So(conn.ReadJSON(&response), ShouldBeNil)
				So(response.Type, ShouldEqual, "error")
				So(response.Payload, ShouldEqual, "unknown message type")
				So(response.Date, ShouldHappenWithin, (time.Second * 5), time.Now())
			})

			Convey("responds with corresponding ProcessorFn", func() {
				hub, conn := testSetup()

				hub.RegisterProcessor("echo", func(hub *Hub, request *Message) (*Message, error) {
					return request, nil
				})

				go hub.listen(conn)

				request := Message{
					Type:    "echo",
					Date:    time.Now(),
					Payload: "repeat me!",
				}

				So(conn.WriteJSON(&request), ShouldBeNil)

				var response Message
				So(conn.ReadJSON(&response), ShouldBeNil)
				So(response, ShouldResemble, request)
			})

			Convey("deals with multiple message types", func() {
				hub, conn := testSetup()

				hub.RegisterProcessor("echo", func(_ *Hub, request *Message) (*Message, error) {
					return request, nil
				})
				hub.RegisterProcessor("hello", func(_ *Hub, request *Message) (*Message, error) {
					str, ok := request.Payload.(string)
					if !ok {
						return nil, fmt.Errorf("Couldn't read name %#v", request.Payload)
					}

					return &Message{
						Type:    "greetings",
						Payload: fmt.Sprintf("hello %s", str),
					}, nil
				})

				go hub.listen(conn)

				request := Message{
					Type:    "echo",
					Date:    time.Now(),
					Payload: "repeat me!",
				}

				So(conn.WriteJSON(&request), ShouldBeNil)

				var response Message
				So(conn.ReadJSON(&response), ShouldBeNil)
				So(response, ShouldResemble, request)

				request2 := Message{
					Type:    "hello",
					Payload: "don",
				}
				So(conn.WriteJSON(&request2), ShouldBeNil)
				var response2 Message
				So(conn.ReadJSON(&response2), ShouldBeNil)
				So(response2.Payload.(string), ShouldEqual, "hello don")
			})

			Convey("forwards errors down the stream", func() {
				hub, conn := testSetup()

				hub.RegisterProcessor("error", func(hub *Hub, request *Message) (*Message, error) {
					return nil, fmt.Errorf("forced error")
				})

				go hub.listen(conn)

				So(conn.WriteJSON(&Message{Type: "error"}), ShouldBeNil)

				var response Message
				So(conn.ReadJSON(&response), ShouldBeNil)
				So(response.Type, ShouldEqual, "error")
				So(response.Payload.(string), ShouldEqual, "forced error")
			})
		})
	})
}
