package bot

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const HeaderTelegramSecretToken = "X-Telegram-Bot-Api-Secret-Token"

func (b *Bot) startWebhook(ctx context.Context) error {
	parsedURL, err := url.Parse(b.cfg.WebhookURL)
	if err != nil {
		return fmt.Errorf("analisar WEBHOOK_URL: %w", err)
	}

	host := parsedURL.Hostname()
	isIP := net.ParseIP(host) != nil

	var tlsCert *tls.Certificate
	var certPEM []byte

	if b.cfg.WebhookCertPath != "" && b.cfg.WebhookKeyPath != "" {
		certBytes, err := os.ReadFile(b.cfg.WebhookCertPath)
		if err != nil {
			return fmt.Errorf("ler arquivo de certificado em %s: %w", b.cfg.WebhookCertPath, err)
		}
		loadedCert, err := tls.LoadX509KeyPair(b.cfg.WebhookCertPath, b.cfg.WebhookKeyPath)
		if err != nil {
			return fmt.Errorf("carregar par de chaves tls (%s, %s): %w", b.cfg.WebhookCertPath, b.cfg.WebhookKeyPath, err)
		}
		tlsCert = &loadedCert
		certPEM = certBytes
	} else if isIP {
		b.logger.Info("gerando certificado autoassinado para IP da VPS...", "ip", host)
		generatedCert, generatedPEM, err := GenerateSelfSignedCert(host)
		if err != nil {
			return fmt.Errorf("gerar certificado autoassinado para IP %s: %w", host, err)
		}
		tlsCert = &generatedCert
		certPEM = generatedPEM
	}

	params := make(tgbotapi.Params)
	params["url"] = b.cfg.WebhookURL
	if b.cfg.WebhookSecretToken != "" {
		params["secret_token"] = b.cfg.WebhookSecretToken
	}
	params.AddBool("drop_pending_updates", false)

	var apiResp *tgbotapi.APIResponse
	if len(certPEM) > 0 {
		files := []tgbotapi.RequestFile{
			{
				Name: "certificate",
				Data: tgbotapi.FileBytes{
					Name:  "cert.pem",
					Bytes: certPEM,
				},
			},
		}
		apiResp, err = b.api.UploadFiles("setWebhook", params, files)
	} else {
		apiResp, err = b.api.MakeRequest("setWebhook", params)
	}

	if err != nil {
		return fmt.Errorf("falha ao registrar webhook no telegram: %w", err)
	}

	b.logger.Info("webhook registrado no telegram com sucesso",
		"url", b.cfg.WebhookURL,
		"has_secret", b.cfg.WebhookSecretToken != "",
		"has_cert", len(certPEM) > 0,
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

	if tlsCert != nil {
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{*tlsCert},
		}
	}

	errChan := make(chan error, 1)
	go func() {
		if tlsCert != nil {
			b.logger.Info("servidor HTTPS de webhook escutando com TLS",
				"addr", server.Addr,
				"path", b.cfg.WebhookPath,
			)
			if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
			}
		} else {
			b.logger.Info("servidor HTTP de webhook escutando",
				"addr", server.Addr,
				"path", b.cfg.WebhookPath,
			)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
			}
		}
	}()

	select {
	case <-ctx.Done():
		b.logger.Info("encerrando servidor de webhook...")
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
