package fsm_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStore_BasicLifecycle(t *testing.T) {
	store := fsm.NewSessionStore(10 * time.Minute)

	// Inicia sem sessão
	_, exists := store.Get(123)
	assert.False(t, exists)

	// Cria sessão
	sess := store.GetOrCreate(123, 456, "Marcos", "marcos_v")
	require.NotNil(t, sess)
	assert.Equal(t, int64(123), sess.TelegramID)
	assert.Equal(t, int64(456), sess.ChatID)
	assert.Equal(t, "Marcos", sess.FirstName)
	assert.Equal(t, fsm.StateIdle, sess.CurrentState)

	// Busca existente
	found, exists := store.Get(123)
	assert.True(t, exists)
	assert.Equal(t, sess, found)

	// Atualiza estado
	sess.CurrentState = fsm.StateWaitingBalance
	sess.InitialBalance = 5000.00
	store.Set(sess)

	updated, exists := store.Get(123)
	assert.True(t, exists)
	assert.Equal(t, fsm.StateWaitingBalance, updated.CurrentState)
	assert.Equal(t, 5000.00, updated.InitialBalance)

	// Deleta sessão
	store.Delete(123)
	_, exists = store.Get(123)
	assert.False(t, exists)
}

func TestSessionStore_EvictExpired(t *testing.T) {
	store := fsm.NewSessionStore(50 * time.Millisecond)

	// Cria sessão que expirará
	sess1 := store.GetOrCreate(1, 100, "User1", "u1")
	sess1.LastActiveAt = time.Now().UTC().Add(-100 * time.Millisecond)
	store.Set(sess1)

	// Cria sessão que ainda está fresca
	sess2 := store.GetOrCreate(2, 200, "User2", "u2")
	sess2.LastActiveAt = time.Now().UTC()
	store.Set(sess2)

	// Executa evicção
	evicted := store.EvictExpired()
	assert.Equal(t, 1, evicted)

	// Verifica que sess1 foi expurgada e sess2 permaneceu
	_, exists1 := store.Get(1)
	assert.False(t, exists1)

	_, exists2 := store.Get(2)
	assert.True(t, exists2)
}

func TestSessionStore_EvictionWorker(t *testing.T) {
	store := fsm.NewSessionStore(20 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicia worker com intervalo bem curto para o teste
	go store.StartEvictionWorker(ctx, 10*time.Millisecond)

	sess := store.GetOrCreate(999, 999, "Ephemeral", "eph")
	sess.LastActiveAt = time.Now().UTC().Add(-50 * time.Millisecond)
	store.Set(sess)

	// Aguarda o worker rodar
	require.Eventually(t, func() bool {
		_, exists := store.Get(999)
		return !exists
	}, 200*time.Millisecond, 10*time.Millisecond, "sessão expirada deveria ser removida pelo worker")
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	store := fsm.NewSessionStore(30 * time.Minute)
	var wg sync.WaitGroup

	numUsers := 50
	operationsPerUser := 100

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			telegramID := int64(1000 + id)
			chatID := int64(2000 + id)

			for op := 0; op < operationsPerUser; op++ {
				switch op % 4 {
				case 0:
					store.GetOrCreate(telegramID, chatID, fmt.Sprintf("User%d", id), "")
				case 1:
					if s, exists := store.Get(telegramID); exists {
						s.InitialBalance += 10.50
						store.Set(s)
					}
				case 2:
					store.Touch(telegramID)
				case 3:
					if op == operationsPerUser-1 {
						store.Delete(telegramID)
					}
				}
			}
		}(i)
	}

	wg.Wait()
}
