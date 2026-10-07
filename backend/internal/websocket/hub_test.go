package websocket

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ws "github.com/gorilla/websocket"
)

func TestSlowSubscriberCannotBlockHealthySubscriber(t *testing.T) {
	makeClient := testClientFactory(t)
	slow, healthy := makeClient(), makeClient()
	healthy.send = make(chan []byte, 64)
	hub := NewHub(2)
	defer hub.Close()
	if err := hub.Subscribe(slow); err != nil {
		t.Fatal(err)
	}
	if err := hub.Subscribe(healthy); err != nil {
		t.Fatal(err)
	}
	for i := range 33 {
		hub.Broadcast([]byte{byte(i)})
	}
	select {
	case <-slow.done:
	default:
		t.Fatal("slow client was not disconnected")
	}
	if hub.Viewers() != 1 {
		t.Fatal("healthy subscriber was affected by a slow peer")
	}
	for i := range 33 {
		select {
		case data := <-healthy.send:
			if data[0] != byte(i) {
				t.Fatal("healthy subscriber lost data ordering")
			}
		default:
			t.Fatal("healthy subscriber lost a chunk")
		}
	}
}

func TestStoppedHubRejectsLateSubscribersAndResumes(t *testing.T) {
	makeClient := testClientFactory(t)
	hub := NewHub(2)
	defer hub.Close()
	first := makeClient()
	if err := hub.Subscribe(first); err != nil {
		t.Fatal(err)
	}
	hub.Stop()
	select {
	case <-first.done:
	default:
		t.Fatal("Stop did not disconnect its viewer")
	}
	late := makeClient()
	if err := hub.Subscribe(late); !errors.Is(err, ErrStreamStopped) {
		t.Fatalf("stopped source accepted a late handshake: %v", err)
	}
	if hub.Viewers() != 0 {
		t.Fatal("stopped source retained a viewer")
	}
	hub.Start()
	if err := hub.Subscribe(late); err != nil {
		t.Fatalf("restarted source rejected its viewer: %v", err)
	}
	hub.Close()
	hub.Start()
	if err := hub.Subscribe(makeClient()); !errors.Is(err, ErrHubClosed) {
		t.Fatal("removed source was reopened")
	}
}

func testClientFactory(t *testing.T) func() *Client {
	t.Helper()
	connections := make(chan *ws.Conn, 2)
	upgrader := ws.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		connections <- conn
	}))
	t.Cleanup(server.Close)
	return func() *Client {
		remote, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = remote.Close() })
		client := NewClient(<-connections)
		t.Cleanup(client.Close)
		return client
	}
}
