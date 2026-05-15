package realtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string][]chan []byte
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string][]chan []byte)}
}

func (h *Hub) Broadcast(weddingID string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("data: %s\n\n", data))
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients[weddingID] {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *Hub) ServeSSE(c *fiber.Ctx) error {
	weddingID := c.Params("id")

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	ch := make(chan []byte, 16)
	h.subscribe(weddingID, ch)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer h.unsubscribe(weddingID, ch)

		if _, err := fmt.Fprintf(w, "event: connected\ndata: {}\n\n"); err != nil {
			return
		}
		_ = w.Flush()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg := <-ch:
				if _, err := w.Write(msg); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}

func (h *Hub) subscribe(weddingID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[weddingID] = append(h.clients[weddingID], ch)
}

func (h *Hub) unsubscribe(weddingID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[weddingID]
	for i, c := range list {
		if c == ch {
			h.clients[weddingID] = append(list[:i], list[i+1:]...)
			break
		}
	}
}
