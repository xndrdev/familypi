// Package hub verteilt Aenderungs-Signale an verbundene Clients (Server-Sent Events).
package hub

import "sync"

type Hub struct {
	mu   sync.Mutex
	subs map[chan string]struct{}
}

func New() *Hub { return &Hub{subs: map[chan string]struct{}{}} }

func (h *Hub) Subscribe() chan string {
	ch := make(chan string, 8)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

// Publish meldet allen Abonnenten, dass sich ein Bereich (z.B. "todos") geaendert hat.
// Langsame Clients verlieren Nachrichten statt den Sender zu blockieren.
func (h *Hub) Publish(topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- topic:
		default:
		}
	}
}
