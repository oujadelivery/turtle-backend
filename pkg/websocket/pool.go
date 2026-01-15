package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Pool struct {
	clients    map[uint]*websocket.Conn
	mu         sync.RWMutex
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	UserID uint
	Conn   *websocket.Conn
}

type Message struct {
	UserID  uint
	Type    string
	Payload interface{}
}

func NewPool() *Pool {
	return &Pool{
		clients:    make(map[uint]*websocket.Conn),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (p *Pool) Run() {
	for {
		select {
		case client := <-p.register:
			p.mu.Lock()
			p.clients[client.UserID] = client.Conn
			p.mu.Unlock()

		case client := <-p.unregister:
			p.mu.Lock()
			delete(p.clients, client.UserID)
			p.mu.Unlock()
			client.Conn.Close()

		case message := <-p.broadcast:
			p.mu.RLock()
			if conn, ok := p.clients[message.UserID]; ok {
				conn.WriteJSON(message)
			}
			p.mu.RUnlock()
		}
	}
}

func (p *Pool) SendToUser(userID uint, messageType string, payload interface{}) {
	p.broadcast <- Message{
		UserID:  userID,
		Type:    messageType,
		Payload: payload,
	}
}
