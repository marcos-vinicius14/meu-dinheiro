package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const HeaderTelegramSecretToken = "X-Telegram-Bot-Api-Secret-Token"

func (b *Bot) startWebhook(ctx context.Context) error {
	params := make(tgbotapi.Params)
	params["url"] = b.cfg.WebhookURL
	if b.cfg.WebhookSecretToken != "" {
		params["secret_token"] = b.cfg.WebhookSecretToken
	}
	params.AddBool("drop_pending_updates", false)

	apiResp, err := b.api.MakeRequest("setWebhook", params)
	if err != nil {
		return fmt.Errorf("falha ao registrar webhook no telegram: %w", err)
	}

	b.logger.Info("webhook registrado no telegram com sucesso",
		"url", b.cfg.WebhookURL,
		"has_secret", b.cfg.WebhookSecretToken != "",
		"description", apiResp.Description,
	)

	mux := b.NewWebhookHandler()
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", b.cfg.WebhookPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		b.logger.Info("servidor HTTP de webhook escutando",
			"addr", server.Addr,
			"path", b.cfg.WebhookPath,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		b.logger.Info("encerrando servidor de webhook HTTP...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return fmt.Errorf("erro no servidor HTTP de webhook: %w", err)
	}
}

func (b *Bot) NewWebhookHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP","mode":"webhook"}`))
	})

	mux.HandleFunc(b.cfg.WebhookPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
			return
		}

		if b.cfg.WebhookSecretToken != "" {
			token := r.Header.Get(HeaderTelegramSecretToken)
			if token != b.cfg.WebhookSecretToken {
				b.logger.Warn("webhook com secret token inválido ou ausente", "remote_addr", r.RemoteAddr)
				http.Error(w, "não autorizado", http.StatusUnauthorized)
				return
			}
		}

		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			b.logger.Error("falha ao decodificar update do telegram", "error", err)
			http.Error(w, "requisição inválida", http.StatusBadRequest)
			return
		}

		b.handleUpdate(r.Context(), &update)

		w.WriteHeader(http.StatusOK)
	})

	return mux
}
