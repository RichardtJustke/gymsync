package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/handler"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/repository"
	"gymsync/backend/internal/service"
	"gymsync/backend/pkg/mockbilling"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Carrega arquivo .env se existir
	_ = godotenv.Load()

	// 1. Inicializa o banco de dados
	db, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Falha crítica ao conectar no banco de dados: %v", err)
	}

	// 2. Executa Seed com 110+ exercícios e usuários padrão de teste
	if err := repository.SeedDatabase(db); err != nil {
		log.Printf("[AVISO] Falha ao executar seed: %v", err)
	}

	// 3. Inicializa Serviços
	importerService := service.NewImporterService(db)
	billingService := mockbilling.NewMockBillingService(db)

	// 4. Inicializa Handlers
	authHandler := handler.NewAuthHandler(db)
	dashHandler := handler.NewDashboardHandler(db)
	studentsHandler := handler.NewStudentsHandler(db, importerService)
	exercisesHandler := handler.NewExercisesHandler(db)
	workoutsHandler := handler.NewWorkoutsHandler(db)
	billingHandler := handler.NewBillingHandler(db, billingService)
	checkInHandler := handler.NewCheckInHandler(db)

	// 5. Configura Roteador Chi
	r := chi.NewRouter()

	// Middlewares globais
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS liberado para o frontend Next.js
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rotas Públicas
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","app":"GymSync API","version":"1.0.0"}`))
	})

	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/register-gym", authHandler.RegisterGym)
	r.Get("/api/invoices/{id}/boleto", billingHandler.RenderBoleto)

	// Rotas Autenticadas
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.AuthMiddleware)

		// Perfil do usuário atual
		protected.Get("/api/auth/me", authHandler.Me)

		// Exercícios (disponíveis para todos os usuários autenticados)
		protected.Get("/api/exercises", exercisesHandler.ListExercises)
		protected.Post("/api/exercises", exercisesHandler.CreateExercise)

		// Check-in
		protected.Post("/api/checkins", checkInHandler.DoCheckIn)
		protected.Get("/api/checkins/today", checkInHandler.GetTodayCheckIns)

		// Mock Payment (Aluno e Gestor)
		protected.Post("/api/invoices/{id}/mock-pay", billingHandler.MockPay)

		// Rotas exclusivas do Aluno
		protected.Get("/api/workouts/my", workoutsHandler.GetMyWorkouts)
		protected.Post("/api/workouts/items/{itemId}/load", workoutsHandler.UpdateExerciseLoad)
		protected.Get("/api/invoices/my", billingHandler.GetMyInvoices)

		// Rotas de Gestão (Admin e Professor)
		protected.Group(func(staff chi.Router) {
			staff.Use(middleware.RequireRoles(domain.RoleAdmin, domain.RoleProfessor))
			staff.Get("/api/students", studentsHandler.ListStudents)
			staff.Get("/api/students/{id}", studentsHandler.GetStudent)
			staff.Get("/api/students/{studentId}/workouts", workoutsHandler.GetStudentWorkouts)
			staff.Post("/api/students/{studentId}/workouts", workoutsHandler.CreateWorkoutPlan)
		})

		// Rotas exclusivas do Gestor da Academia (Admin)
		protected.Group(func(admin chi.Router) {
			admin.Use(middleware.RequireRoles(domain.RoleAdmin))
			admin.Get("/api/dashboard/stats", dashHandler.GetStats)
			admin.Post("/api/students", studentsHandler.CreateStudent)
			admin.Post("/api/students/import/preview", studentsHandler.ImportPreview)
			admin.Post("/api/students/import/execute", studentsHandler.ExecuteImport)
			admin.Get("/api/invoices", billingHandler.ListInvoices)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("🚀 GymSync API rodando na porta %s (http://localhost:%s)", port, port)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Falha ao iniciar servidor HTTP: %v", err)
	}
}
