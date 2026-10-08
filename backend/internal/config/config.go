package config

import (
	"log"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // Garante o fuso America/Sao_Paulo mesmo em containers mínimos
)

// Config reúne as configurações lidas do ambiente
type Config struct {
	Env            string // development | production
	Port           string
	JWTSecret      []byte
	AllowedOrigins []string
	PublicGymID    string // Academia usada pelas rotas públicas do site (cadastro de leads)
	FirebaseCreds  string // Caminho do JSON da service account do Firebase (push)
	WebhookSecret  string // Chave HMAC dos webhooks de pagamento (POST /api/webhooks/payments)
	SeedDemo       bool
	Location       *time.Location
}

var C Config

// Default origens do painel em dev + WebViews do Capacitor (Android usa https://localhost, iOS capacitor://localhost)
const defaultOrigins = "http://localhost:3010,http://localhost:3000,https://localhost,http://localhost,capacitor://localhost"

func Load() Config {
	env := getenv("APP_ENV", "development")
	c := Config{
		Env:           env,
		Port:          getenv("PORT", "8080"),
		PublicGymID:   os.Getenv("PUBLIC_GYM_ID"),
		FirebaseCreds: os.Getenv("FIREBASE_CREDENTIALS_FILE"),
		WebhookSecret: os.Getenv("PAYMENT_WEBHOOK_SECRET"),
		SeedDemo:      getenv("SEED_DEMO", map[bool]string{true: "false", false: "true"}[env == "production"]) == "true",
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		if c.IsProduction() {
			log.Fatal("JWT_SECRET é obrigatório em produção")
		}
		log.Println("[CONFIG] JWT_SECRET não definido — usando chave de desenvolvimento. NUNCA use assim em produção.")
		secret = "dev-only-insecure-secret-unika"
	}
	if c.IsProduction() && len(secret) < 32 {
		log.Fatal("JWT_SECRET deve ter pelo menos 32 caracteres")
	}
	c.JWTSecret = []byte(secret)

	for _, o := range strings.Split(getenv("ALLOWED_ORIGINS", defaultOrigins), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}

	loc, err := time.LoadLocation(getenv("GYM_TIMEZONE", "America/Sao_Paulo"))
	if err != nil {
		log.Fatalf("GYM_TIMEZONE inválido: %v", err)
	}
	c.Location = loc

	C = c
	return c
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Now retorna o horário atual no fuso da academia
func Now() time.Time { return time.Now().In(loc()) }

// StartOfDay meia-noite (no fuso da academia) do dia de t
func StartOfDay(t time.Time) time.Time {
	t = t.In(loc())
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc())
}

func loc() *time.Location {
	if C.Location == nil {
		l, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			return time.UTC
		}
		return l
	}
	return C.Location
}

// Loc expõe o fuso da academia
func Loc() *time.Location { return loc() }
