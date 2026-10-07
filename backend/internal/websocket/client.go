package websocket

import (
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

type Client struct {
	conn *ws.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
}

func NewClient(conn *ws.Conn) *Client {
	return &Client{conn: conn, send: make(chan []byte, 32), done: make(chan struct{})}
}

// Channels carrying data are never closed; done handles cancellation without send/close races.
func (c *Client) Close() { c.once.Do(func() { close(c.done); _ = c.conn.Close() }) }

func (c *Client) Run() {
	writerDone := make(chan struct{})
	go func() { defer close(writerDone); c.writePump() }()
	c.readPump()
	c.Close()
	<-writerDone
}

func (c *Client) readPump() {
	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	defer c.Close()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case data := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(ws.BinaryMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(ws.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}
