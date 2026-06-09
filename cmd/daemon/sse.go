package main

import (
	"context"
	"sync"
)

// SSEBroker manages Server-Sent Events client subscriptions and broadcasts.
type SSEBroker struct {
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
	mu         sync.RWMutex
	lastStatus []byte
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 16),
	}
}

// Run starts the broker event loop. Blocks until ctx is cancelled.
func (b *SSEBroker) Run(ctx context.Context) {
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			// Send cached status to new clients immediately
			if b.lastStatus != nil {
				select {
				case client <- b.lastStatus:
				default:
				}
			}
			b.mu.Unlock()

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
			}
			b.mu.Unlock()

		case msg := <-b.broadcast:
			b.mu.RLock()
			b.lastStatus = msg
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					// Client too slow — drop it
					close(client)
					delete(b.clients, client)
				}
			}
			b.mu.RUnlock()

		case <-ctx.Done():
			return
		}
	}
}

// Publish sends a status update to all connected SSE clients.
func (b *SSEBroker) Publish(status []byte) {
	select {
	case b.broadcast <- status:
	default:
	}
}

var broker *SSEBroker
