package session

import (
	"sync"
	"testing"
	"time"
)

func TestSessionManager_Debounce(t *testing.T) {
	var receivedMessage string
	var receivedPhone string
	var mu sync.Mutex

	onDebounce := func(phone, profileName, msg string) {
		mu.Lock()
		defer mu.Unlock()
		receivedPhone = phone
		receivedMessage = msg
	}

	sm := NewSessionManager(50*time.Millisecond, onDebounce)

	// Send 4 fragmented messages in quick succession (< 50ms)
	sm.AddMessage("5491112345678", "Juan", "Hola")
	time.Sleep(10 * time.Millisecond)
	sm.AddMessage("5491112345678", "Juan", "quiero pedir")
	time.Sleep(10 * time.Millisecond)
	sm.AddMessage("5491112345678", "Juan", "2 combos de Nutella")
	time.Sleep(10 * time.Millisecond)
	sm.AddMessage("5491112345678", "Juan", "para retirar")

	// Wait for debounce timer to fire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedPhone != "5491112345678" {
		t.Fatalf("expected phone 5491112345678, got %s", receivedPhone)
	}

	expected := "Hola\nquiero pedir\n2 combos de Nutella\npara retirar"
	if receivedMessage != expected {
		t.Fatalf("expected combined message:\n%s\ngot:\n%s", expected, receivedMessage)
	}
}

func TestSessionManager_History(t *testing.T) {
	sm := NewSessionManager(50*time.Millisecond, nil)
	phone := "5491199998888"

	sm.AddMessage(phone, "Maria", "Hola")
	hist := sm.GetHistory(phone)
	if hist == nil {
		t.Fatalf("expected history slice, got nil")
	}

	sm.ClearSession(phone)
	if len(sm.GetHistory(phone)) != 0 {
		t.Fatalf("expected cleared history")
	}
}
