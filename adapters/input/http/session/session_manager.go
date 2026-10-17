package session

import (
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

type UserSession struct {
	Phone       string
	ProfileName string
	Buffer      []string
	Timer       *time.Timer
	History     []*genai.Content
	LastActive  time.Time
}

type MessageHandler func(phone, profileName, combinedMessage string)

type SessionManager struct {
	mu             sync.RWMutex
	sessions       map[string]*UserSession
	debounceWindow time.Duration
	onDebounce     MessageHandler
}

func NewSessionManager(debounceWindow time.Duration, onDebounce MessageHandler) *SessionManager {
	if debounceWindow <= 0 {
		debounceWindow = 4 * time.Second
	}
	sm := &SessionManager{
		sessions:       make(map[string]*UserSession),
		debounceWindow: debounceWindow,
		onDebounce:     onDebounce,
	}

	// Inactivity cleaner every 30 minutes
	go sm.startCleaner()

	return sm
}

func (sm *SessionManager) AddMessage(phone, profileName, text string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, exists := sm.sessions[phone]
	if !exists {
		s = &UserSession{
			Phone:       phone,
			ProfileName: profileName,
			Buffer:      make([]string, 0),
			History:     make([]*genai.Content, 0),
			LastActive:  time.Now(),
		}
		sm.sessions[phone] = s
	}

	if profileName != "" {
		s.ProfileName = profileName
	}
	s.LastActive = time.Now()
	s.Buffer = append(s.Buffer, strings.TrimSpace(text))

	// Stop previous timer if running
	if s.Timer != nil {
		s.Timer.Stop()
	}

	// Start / reset debounce timer
	s.Timer = time.AfterFunc(sm.debounceWindow, func() {
		sm.flushBuffer(phone)
	})
}

func (sm *SessionManager) flushBuffer(phone string) {
	sm.mu.Lock()
	s, exists := sm.sessions[phone]
	if !exists || len(s.Buffer) == 0 {
		sm.mu.Unlock()
		return
	}

	combined := strings.Join(s.Buffer, "\n")
	profileName := s.ProfileName
	s.Buffer = make([]string, 0)
	sm.mu.Unlock()

	if sm.onDebounce != nil && strings.TrimSpace(combined) != "" {
		sm.onDebounce(phone, profileName, combined)
	}
}

func (sm *SessionManager) GetHistory(phone string) []*genai.Content {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, exists := sm.sessions[phone]
	if !exists {
		return nil
	}
	// Return a copy
	out := make([]*genai.Content, len(s.History))
	copy(out, s.History)
	return out
}

func (sm *SessionManager) SetHistory(phone string, history []*genai.Content) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, exists := sm.sessions[phone]
	if !exists {
		s = &UserSession{
			Phone:      phone,
			History:    history,
			LastActive: time.Now(),
		}
		sm.sessions[phone] = s
		return
	}
	s.History = history
	s.LastActive = time.Now()
}

func (sm *SessionManager) AppendHistory(phone string, content *genai.Content) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, exists := sm.sessions[phone]
	if exists {
		s.History = append(s.History, content)
		s.LastActive = time.Now()
	}
}

func (sm *SessionManager) ClearSession(phone string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, exists := sm.sessions[phone]; exists {
		if s.Timer != nil {
			s.Timer.Stop()
		}
		delete(sm.sessions, phone)
	}
}

func (sm *SessionManager) startCleaner() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		cutoff := time.Now().Add(-2 * time.Hour)
		for phone, session := range sm.sessions {
			if session.LastActive.Before(cutoff) {
				if session.Timer != nil {
					session.Timer.Stop()
				}
				delete(sm.sessions, phone)
			}
		}
		sm.mu.Unlock()
	}
}
