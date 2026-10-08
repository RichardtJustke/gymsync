package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// PushMessage conteúdo de uma notificação push
type PushMessage struct {
	Title string
	Body  string
	Link  string
}

// PushSender envia push para tokens de aparelhos. Retorna os tokens inválidos (para remoção).
type PushSender interface {
	Send(ctx context.Context, tokens []string, msg PushMessage) (invalid []string)
}

// LogPushSender usado quando o Firebase não está configurado (dev): só registra no log
type LogPushSender struct{}

func (LogPushSender) Send(_ context.Context, tokens []string, msg PushMessage) []string {
	log.Printf("[PUSH:dev] %d aparelho(s) ← %q: %s", len(tokens), msg.Title, msg.Body)
	return nil
}

// FCMSender envia via Firebase Cloud Messaging HTTP v1 (Android e iOS/APNs)
type FCMSender struct {
	projectID string
	ts        oauth2.TokenSource
	client    *http.Client
}

// NewPushSender cria o sender do FCM a partir do JSON da service account; sem arquivo, usa o de log
func NewPushSender(credsFile string) PushSender {
	if credsFile == "" {
		log.Println("[PUSH] FIREBASE_CREDENTIALS_FILE não definido — notificações push só serão registradas no log")
		return LogPushSender{}
	}
	data, err := os.ReadFile(credsFile)
	if err != nil {
		log.Fatalf("[PUSH] Falha ao ler credenciais do Firebase: %v", err)
	}
	creds, err := google.CredentialsFromJSON(context.Background(), data, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil || creds.ProjectID == "" {
		log.Fatalf("[PUSH] Credenciais do Firebase inválidas: %v", err)
	}
	log.Printf("[PUSH] Firebase Cloud Messaging ativo (projeto %s)", creds.ProjectID)
	return &FCMSender{projectID: creds.ProjectID, ts: creds.TokenSource, client: &http.Client{Timeout: 10 * time.Second}}
}

func (f *FCMSender) Send(ctx context.Context, tokens []string, msg PushMessage) []string {
	tok, err := f.ts.Token()
	if err != nil {
		log.Printf("[PUSH] Falha ao obter token OAuth do Firebase: %v", err)
		return nil
	}
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)

	var invalid []string
	for _, t := range tokens {
		payload := map[string]any{
			"message": map[string]any{
				"token":        t,
				"notification": map[string]string{"title": msg.Title, "body": msg.Body},
				"data":         map[string]string{"link": msg.Link},
				"android":      map[string]any{"priority": "high", "notification": map[string]string{"sound": "default"}},
				"apns":         map[string]any{"payload": map[string]any{"aps": map[string]any{"sound": "default"}}},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

		res, err := f.client.Do(req)
		if err != nil {
			log.Printf("[PUSH] Erro de rede ao enviar push: %v", err)
			continue
		}
		respBody, _ := io.ReadAll(res.Body)
		res.Body.Close()
		switch {
		case res.StatusCode == http.StatusOK:
		case res.StatusCode == http.StatusNotFound || bytes.Contains(respBody, []byte("UNREGISTERED")) || bytes.Contains(respBody, []byte("INVALID_ARGUMENT")):
			invalid = append(invalid, t)
		default:
			log.Printf("[PUSH] FCM respondeu %d: %s", res.StatusCode, string(respBody))
		}
	}
	return invalid
}
