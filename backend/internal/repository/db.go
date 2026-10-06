package repository

import (
	"fmt"
	"log"
	"os"

	"gymsync/backend/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB inicializa o banco de dados (PostgreSQL com fallback para SQLite local se não houver conexão ativa)
func InitDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "5432")
		user := getEnvOrDefault("DB_USER", "postgres")
		pass := getEnvOrDefault("DB_PASSWORD", "postgrespassword")
		dbname := getEnvOrDefault("DB_NAME", "gymsync")
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Sao_Paulo",
			host, user, pass, dbname, port)
	}

	var db *gorm.DB
	var err error

	// Tenta conectar no PostgreSQL
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		log.Printf("[DB] Falha ao conectar no PostgreSQL (%v). Usando SQLite local para desenvolvimento contínuo...", err)
		db, err = gorm.Open(sqlite.Open("gymsync.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			return nil, fmt.Errorf("falha ao inicializar banco de dados SQLite: %w", err)
		}
		log.Println("[DB] Conectado com sucesso ao SQLite local (gymsync.db).")
	} else {
		log.Println("[DB] Conectado com sucesso ao PostgreSQL.")
	}

	// Executa AutoMigrate de todas as entidades
	err = db.AutoMigrate(
		&domain.Gym{},
		&domain.User{},
		&domain.Student{},
		&domain.Exercise{},
		&domain.WorkoutPlan{},
		&domain.Workout{},
		&domain.WorkoutExercise{},
		&domain.Invoice{},
		&domain.CheckIn{},
	)
	if err != nil {
		return nil, fmt.Errorf("erro durante migração de esquema: %w", err)
	}

	DB = db
	return db, nil
}

func getEnvOrDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
