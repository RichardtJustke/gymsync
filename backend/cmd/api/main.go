package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/handler"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/repository"
	"gymsync/backend/internal/service"
	"gymsync/backend/pkg/mockbilling"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	db, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Falha crítica ao conectar no banco de dados: %v", err)
	}
	if err := repository.SeedDatabase(db); err != nil {
		log.Printf("[AVISO] Falha ao executar seed: %v", err)
	}

	// Serviços
	notifier := service.NewNotifier(db, service.NewPushSender(cfg.FirebaseCreds))
	perms := service.NewPermissions(db)
	agenda := service.NewAgendaService(db, notifier)
	codes := service.NewCheckinCodes(cfg.JWTSecret)
	importer := service.NewImporterService(db)
	// Gateway de pagamento: simulado até integrar um provedor real (ver service.PaymentGateway)
	billing := service.NewBillingService(db, mockbilling.MockGateway{}, notifier)
	team := service.NewTeamService(db)
	service.NewScheduler(db, notifier, agenda, billing).Start()

	// Handlers
	authH := handler.NewAuthHandler(db, perms)
	dashH := handler.NewDashboardHandler(db, billing, perms)
	studentsH := handler.NewStudentsHandler(db, importer, billing, team)
	exercisesH := handler.NewExercisesHandler(db)
	workoutsH := handler.NewWorkoutsHandler(db, notifier)
	checkInH := handler.NewCheckInHandler(db, codes)
	catalogH := handler.NewCatalogHandler(db)
	agendaH := handler.NewAgendaHandler(db, agenda, notifier)
	leadsH := handler.NewLeadsHandler(db, agenda, notifier, perms, billing, team)
	notifH := handler.NewNotificationsHandler(db, notifier)
	financeH := handler.NewFinanceHandler(db, billing, perms)
	staffH := handler.NewStaffHandler(db, perms, team)
	salesH := handler.NewSalesHandler(db, team, perms)
	auditH := handler.NewAuditHandler(db)
	equipH := handler.NewEquipmentHandler(db)
	templatesH := handler.NewTemplatesHandler(db)
	assessH := handler.NewAssessmentsHandler(db)
	lgpdH := handler.NewLGPDHandler(db)
	webhookH := handler.NewWebhookHandler(db, billing, cfg.WebhookSecret)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:         300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","app":"Unika API","version":"2.1.0"}`))
	})

	// JSON comum: até 1MB (a importação de planilha tem limite próprio)
	jsonLimit := middleware.LimitBody(1 << 20)
	need := func(p ...domain.Permission) func(http.Handler) http.Handler {
		return middleware.RequirePerm(perms, p...)
	}

	r.Route("/api", func(api chi.Router) {
		// ---------------- Rotas públicas ----------------
		// Webhook do gateway: autenticado por HMAC, não por JWT
		api.With(middleware.LimitBody(1<<18)).Post("/webhooks/payments", webhookH.Payments)
		api.With(jsonLimit, httprate.LimitByIP(10, time.Minute)).Post("/auth/login", authH.Login)

		// Integração com o site da Unika (cadastro de interessadas e aula experimental)
		api.Route("/public", func(pub chi.Router) {
			pub.Use(httprate.LimitByIP(60, time.Minute))
			pub.Get("/gym", leadsH.PublicGymInfo)
			pub.Get("/trial-sessions", leadsH.PublicTrialSessions)
			pub.With(jsonLimit, httprate.LimitByIP(5, 10*time.Minute)).Post("/leads", leadsH.PublicCreateLead)
		})

		// ---------------- Rotas autenticadas ----------------
		api.Group(func(p chi.Router) {
			p.Use(middleware.Auth(db))

			p.Group(func(j chi.Router) {
				j.Use(jsonLimit)

				// Conta
				j.Get("/auth/me", authH.Me)
				j.With(httprate.LimitByIP(10, time.Minute)).Post("/auth/password", authH.ChangePassword)
				j.With(httprate.LimitByIP(5, time.Minute)).Delete("/auth/me", authH.DeleteAccount)

				// LGPD e Privacidade (Art. 18 - Portabilidade, Consentimento, Anonimização)
				j.Get("/me/export-data", lgpdH.ExportMyData)
				j.Get("/me/lgpd-consent", lgpdH.GetMyConsent)
				j.Post("/me/lgpd-consent", lgpdH.RegisterConsent)
				j.With(httprate.LimitByIP(5, time.Minute)).Post("/me/lgpd-anonymize", lgpdH.AnonymizeMyData)

				// Notificações (todos)
				j.Get("/notifications/my", notifH.MyNotifications)
				j.Post("/notifications/read", notifH.MarkRead)
				j.Post("/devices", notifH.RegisterDevice)
				j.Delete("/devices", notifH.UnregisterDevice)

				// Consulta comum
				j.Get("/exercises", exercisesH.ListExercises)
				j.Get("/modalities", catalogH.ListModalities)
				j.Get("/sessions", agendaH.ListSessions)

				// Cobranças: aluna dona ou equipe do financeiro (o handler confere)
				j.Get("/invoices/{id}/boleto", financeH.Boleto)
				j.Post("/invoices/{id}/mock-pay", financeH.MockPay)

				// Carga do exercício: aluna (dona da ficha) ou quem prescreve treino
				j.With(middleware.RequireAlunoOrPerm(perms, domain.PermWorkouts)).Post("/workouts/items/{itemId}/load", workoutsH.UpdateExerciseLoad)

				// ---------- App da aluna ----------
				j.Group(func(a chi.Router) {
					a.Use(middleware.RequireRoles(domain.RoleAluno))
					a.Put("/me/profile", studentsH.UpdateMyProfile)
					a.Get("/workouts/my", workoutsH.GetMyWorkouts)
					a.Get("/bookings/my", agendaH.MyBookings)
					a.Post("/sessions/{id}/book", agendaH.Book)
					a.Delete("/sessions/{id}/book", agendaH.CancelBooking)
					a.With(httprate.LimitByIP(10, time.Minute)).Post("/checkins", checkInH.DoCheckIn)
					a.Get("/checkins/my", checkInH.GetMyCheckIns)
					a.Get("/invoices/my", financeH.MyInvoices)
					a.Get("/assessments/my", assessH.GetMyAssessments)
					a.Post("/assessments/my", assessH.CreateMyAssessment)
				})

				// ---------- Equipe (cada rota exige a permissão da área) ----------
				j.Group(func(s chi.Router) {
					s.Use(middleware.RequireStaff)

					// Lista de equipe/professores para selects (qualquer pessoa da equipe)
					s.Get("/teachers", catalogH.ListTeachers)
					s.Get("/plans", catalogH.ListPlans)

					s.With(need(domain.PermDashboard)).Get("/dashboard/stats", dashH.GetStats)
					s.With(need(domain.PermReports)).Get("/reports/frequency", dashH.GetFrequency)
					s.With(need(domain.PermReports)).Get("/reports/overview", dashH.GetOverview)

					// Alunos
					s.With(need(domain.PermStudentsView, domain.PermStudentsEdit)).Get("/students", studentsH.ListStudents)
					s.With(need(domain.PermStudentsView, domain.PermStudentsEdit)).Get("/students/{id}", studentsH.GetStudent)
					s.With(need(domain.PermStudentsEdit)).Post("/students", studentsH.CreateStudent)
					s.With(need(domain.PermStudentsEdit)).Put("/students/{id}", studentsH.UpdateStudent)
					s.With(need(domain.PermStudentsEdit)).Post("/students/{id}/reset-password", studentsH.ResetPassword)
					s.With(need(domain.PermStudentsEdit)).Post("/students/import/execute", studentsH.ExecuteImport)

					// Treinos e Modelos
					s.With(need(domain.PermWorkouts, domain.PermStudentsView)).Get("/students/{studentId}/workouts", workoutsH.GetStudentWorkouts)
					s.With(need(domain.PermWorkouts)).Post("/students/{studentId}/workouts", workoutsH.CreateWorkoutPlan)
					s.With(need(domain.PermWorkouts)).Post("/exercises", exercisesH.CreateExercise)
					s.With(need(domain.PermWorkouts)).Get("/workout-templates", templatesH.ListTemplates)
					s.With(need(domain.PermWorkouts)).Get("/workout-templates/{id}", templatesH.GetTemplate)
					s.With(need(domain.PermWorkouts)).Post("/workout-templates", templatesH.CreateTemplate)
					s.With(need(domain.PermWorkouts)).Delete("/workout-templates/{id}", templatesH.DeleteTemplate)

					// Avaliações Físicas
					s.With(need(domain.PermWorkouts, domain.PermStudentsView)).Get("/students/{studentId}/assessments", assessH.ListStudentAssessments)
					s.With(need(domain.PermWorkouts, domain.PermStudentsEdit)).Post("/students/{studentId}/assessments", assessH.CreateAssessment)
					s.With(need(domain.PermWorkouts, domain.PermStudentsEdit)).Delete("/students/{studentId}/assessments/{assessmentId}", assessH.DeleteAssessment)

					// Recepção
					s.With(need(domain.PermCheckin, domain.PermDashboard)).Get("/checkins/today", checkInH.GetTodayCheckIns)
					s.With(need(domain.PermCheckin)).Get("/checkins/qr", checkInH.GetQRCode)
					s.With(need(domain.PermCheckin)).Post("/checkins/manual", checkInH.ManualCheckIn)

					// Agenda
					s.With(need(domain.PermAgendaView, domain.PermAgendaManage)).Get("/sessions/{id}/attendees", agendaH.SessionAttendees)
					s.With(need(domain.PermAgendaView, domain.PermAgendaManage)).Patch("/bookings/{id}", agendaH.MarkAttendance)
					s.Group(func(ag chi.Router) {
						ag.Use(need(domain.PermAgendaManage))
						ag.Get("/schedules", agendaH.ListSchedules)
						ag.Post("/schedules", agendaH.CreateSchedule)
						ag.Put("/schedules/{id}", agendaH.UpdateSchedule)
						ag.Delete("/schedules/{id}", agendaH.DeleteSchedule)
						ag.Post("/sessions", agendaH.CreateSession)
						ag.Put("/sessions/{id}", agendaH.UpdateSession)
						ag.Post("/sessions/{id}/cancel", agendaH.CancelSession)
					})

					// Leads
					s.Group(func(l chi.Router) {
						l.Use(need(domain.PermLeads))
						l.Get("/leads", leadsH.ListLeads)
						l.Post("/leads", leadsH.CreateLead)
						l.Put("/leads/{id}", leadsH.UpdateLead)
						l.With(need(domain.PermStudentsEdit, domain.PermSales)).Post("/leads/{id}/convert", leadsH.ConvertLead)
					})

					// Vendas e metas
					s.With(need(domain.PermSales, domain.PermGoals)).Get("/sales", salesH.ListSales)
					s.With(need(domain.PermSales, domain.PermGoals)).Get("/sales/ranking", salesH.Ranking)
					s.With(need(domain.PermSales, domain.PermGoals)).Post("/sales", salesH.CreateSale)
					s.With(need(domain.PermGoals)).Post("/sales/{id}/reverse", salesH.ReverseSale)
					s.With(need(domain.PermGoals)).Put("/goals", salesH.SetGoal)

					// Financeiro
					s.With(need(domain.PermFinanceView)).Get("/invoices", financeH.ListInvoices)
					s.With(need(domain.PermFinanceView)).Get("/finance/summary", financeH.Summary)
					s.Group(func(f chi.Router) {
						f.Use(need(domain.PermFinanceManage))
						f.Post("/invoices", financeH.CreateInvoice)
						f.Post("/invoices/generate", financeH.Generate)
						f.Post("/invoices/{id}/pay", financeH.PayInvoice)
						f.Post("/invoices/{id}/cancel", financeH.CancelInvoice)
					})

					// Configurações
					s.Group(func(c chi.Router) {
						c.Use(need(domain.PermCatalog))
						c.Post("/plans", catalogH.CreatePlan)
						c.Put("/plans/{id}", catalogH.UpdatePlan)
						c.Post("/modalities", catalogH.CreateModality)
						c.Put("/modalities/{id}", catalogH.UpdateModality)
						c.Put("/gym", catalogH.UpdateGym)
					})
					s.With(need(domain.PermCatalog, domain.PermDashboard)).Get("/gym", catalogH.GetGym)
					s.With(need(domain.PermAnnounce)).Get("/announcements", notifH.ListAnnouncements)
					s.With(need(domain.PermAnnounce)).Post("/announcements", notifH.CreateAnnouncement)
					s.Group(func(e chi.Router) {
						e.Use(need(domain.PermEquipment))
						e.Get("/equipment", equipH.List)
						e.Post("/equipment", equipH.Create)
						e.Put("/equipment/{id}", equipH.Update)
						e.Delete("/equipment/{id}", equipH.Delete)
					})

					// Equipe, permissões e folha
					s.Group(func(t chi.Router) {
						t.Use(need(domain.PermStaff))
						t.Get("/staff", staffH.ListStaff)
						t.Post("/staff", staffH.CreateStaff)
						t.Put("/staff/{id}", staffH.UpdateStaff)
						t.Post("/staff/{id}/deactivate", staffH.SetActive(false))
						t.Post("/staff/{id}/activate", staffH.SetActive(true))
						t.Post("/staff/{id}/reset-password", staffH.ResetStaffPassword)
						t.Put("/staff/{id}/payroll", staffH.SavePayroll)
						t.Get("/permissions", staffH.GetPermissions)
						t.Put("/permissions/{role}", staffH.SetPermissions)
						t.Get("/payroll", staffH.Payroll)
						// Compatibilidade: cadastro rápido de professor(a)
						t.Post("/teachers", catalogH.CreateTeacher)
						t.Put("/teachers/{id}", catalogH.UpdateTeacher)
						t.Delete("/teachers/{id}", catalogH.DeleteTeacher)
					})
					s.With(need(domain.PermStaff, domain.PermAgendaManage)).Get("/staff/hours", staffH.Hours)
					s.With(need(domain.PermAudit)).Get("/audit", auditH.ListAudit)
				})
			})

			// Upload de planilha (multipart, até 16MB)
			p.With(middleware.RequirePerm(perms, domain.PermStudentsEdit), middleware.LimitBody(16<<20)).
				Post("/students/import/preview", studentsH.ImportPreview)
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("🚀 Unika API (%s) rodando em http://localhost:%s", cfg.Env, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Falha ao iniciar servidor HTTP: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Servidor encerrado")
}
