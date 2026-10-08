package repository

import (
	"fmt"
	"log"
	"os"
	"time"

	"gymsync/backend/internal/config"
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

	if err != nil && config.C.IsProduction() {
		return nil, fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
	}
	if err != nil {
		log.Printf("[DB] Falha ao conectar no PostgreSQL (%v). Usando SQLite local (apenas desenvolvimento)...", err)
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
		&domain.Plan{},
		&domain.Modality{},
		&domain.ClassSchedule{},
		&domain.ClassSession{},
		&domain.Booking{},
		&domain.Lead{},
		&domain.Notification{},
		&domain.Announcement{},
		&domain.DeviceToken{},
		&domain.RolePermission{},
		&domain.StaffProfile{},
		&domain.AuditLog{},
		&domain.Sale{},
		&domain.Goal{},
		&domain.PayrollEntry{},
		&domain.Equipment{},
		&domain.WorkoutTemplate{},
		&domain.WorkoutTemplateDiv{},
		&domain.WorkoutTemplateItem{},
		&domain.PhysicalAssessment{},
		&domain.LGPDConsent{},
		&domain.WebhookEvent{},
	)
	if err != nil {
		return nil, fmt.Errorf("erro durante migração de esquema: %w", err)
	}

	// Pool de conexões otimizado para alta escala (200k+ requisições/dia)
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(60)
		sqlDB.SetMaxIdleConns(30)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
		sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	}

	// Se SQLite local, ativa Write-Ahead Logging (WAL) para permitir leitura concorrente sem bloqueios
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA synchronous = NORMAL;")
	db.Exec("PRAGMA busy_timeout = 5000;")
	db.Exec("PRAGMA cache_size = -64000;") // 64MB cache em memória

	// Criação de índices compostos para tabelas de alta frequência
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_checkins_gym_date ON check_ins (gym_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_checkins_student_date ON check_ins (student_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_students_gym_status ON students (gym_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_gym_status_due ON invoices (gym_id, status, due_date)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_student_status ON invoices (student_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_workout_plans_student_active ON workout_plans (student_id, active)",
		"CREATE INDEX IF NOT EXISTS idx_assessments_student_date ON physical_assessments (student_id, date)",
		"CREATE INDEX IF NOT EXISTS idx_class_sessions_gym_date ON class_sessions (gym_id, date)",
		"CREATE INDEX IF NOT EXISTS idx_lgpd_user ON lgpd_consents (user_id)",
	}
	for _, q := range indexes {
		_ = db.Exec(q)
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
