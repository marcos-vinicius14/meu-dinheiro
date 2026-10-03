package fsm

import (
	"context"
	"sync"
	"time"
)

// SessionStore gerencia o ciclo de vida e concorrência das sessões de onboarding em memória.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
	ttl      time.Duration
}

// NewSessionStore cria um novo SessionStore configurado com um tempo de expiração por inatividade.
func NewSessionStore(ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &SessionStore{
		sessions: make(map[int64]*Session),
		ttl:      ttl,
	}
}

// Get recupera uma cópia por ponteiro da sessão do usuário se existir.
func (s *SessionStore) Get(telegramID int64) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[telegramID]
	return session, exists
}

// GetOrCreate busca a sessão existente ou inicializa uma nova sessão em StateIdle.
func (s *SessionStore) GetOrCreate(telegramID int64, chatID int64, firstName, username string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[telegramID]; exists {
		session.LastActiveAt = time.Now().UTC()
		return session
	}

	newSession := &Session{
		TelegramID:   telegramID,
		ChatID:       chatID,
		FirstName:    firstName,
		Username:     username,
		CurrentState: StateIdle,
		LastActiveAt: time.Now().UTC(),
	}
	s.sessions[telegramID] = newSession
	return newSession
}

// Set atualiza a sessão do usuário protegida por lock.
func (s *SessionStore) Set(session *Session) {
	if session == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if session.LastActiveAt.IsZero() {
		session.LastActiveAt = time.Now().UTC()
	}
	s.sessions[session.TelegramID] = session
}

// Delete remove a sessão do usuário da memória imediatamente.
func (s *SessionStore) Delete(telegramID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, telegramID)
}

// Touch atualiza o timestamp de atividade da sessão.
func (s *SessionStore) Touch(telegramID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[telegramID]; exists {
		session.LastActiveAt = time.Now().UTC()
	}
}

// EvictExpired realiza a limpeza de sessões inativas há mais tempo que o TTL configurado.
// Retorna a quantidade de sessões expurgadas.
func (s *SessionStore) EvictExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	evicted := 0

	for id, session := range s.sessions {
		if now.Sub(session.LastActiveAt) > s.ttl {
			delete(s.sessions, id)
			evicted++
		}
	}
	return evicted
}

// StartEvictionWorker inicia um worker em segundo plano que executa EvictExpired periodicamente.
func (s *SessionStore) StartEvictionWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.EvictExpired()
		}
	}
}
