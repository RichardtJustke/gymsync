package service

import (
	"testing"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fixture struct {
	db      *gorm.DB
	agenda  *AgendaService
	gym     domain.Gym
	session domain.ClassSession
}

func setup(t *testing.T, capacity int, startsIn time.Duration) *fixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Gym{}, &domain.User{}, &domain.Student{}, &domain.Plan{}, &domain.Modality{},
		&domain.ClassSchedule{}, &domain.ClassSession{}, &domain.Booking{}, &domain.Lead{}, &domain.Notification{}, &domain.DeviceToken{}); err != nil {
		t.Fatal(err)
	}
	gym := domain.Gym{Name: "Teste", Document: "1", BookingWindowDays: 7, CancelDeadlineHours: 2}
	db.Create(&gym)
	mod := domain.Modality{GymID: gym.ID, Name: "Pilates", AllowTrial: true, Active: true}
	db.Create(&mod)
	start := time.Now().Add(startsIn)
	s := domain.ClassSession{GymID: gym.ID, ModalityID: mod.ID, StartsAt: start, EndsAt: start.Add(time.Hour), Capacity: capacity, Status: domain.SessionScheduled}
	db.Create(&s)
	notifier := NewNotifier(db, LogPushSender{})
	return &fixture{db: db, agenda: NewAgendaService(db, notifier), gym: gym, session: s}
}

func (f *fixture) student(t *testing.T, plan *domain.Plan) uuid.UUID {
	t.Helper()
	u := domain.User{GymID: f.gym.ID, Name: "Aluna", Email: uuid.NewString() + "@x.com", PasswordHash: "x", Role: domain.RoleAluno}
	f.db.Create(&u)
	st := domain.Student{GymID: f.gym.ID, UserID: u.ID, Status: domain.StatusAtivo}
	if plan != nil {
		st.PlanID = &plan.ID
	}
	f.db.Create(&st)
	return st.ID
}

func TestBookFillsCapacityThenWaitlist(t *testing.T) {
	f := setup(t, 1, 24*time.Hour)
	a, b := f.student(t, nil), f.student(t, nil)

	first, err := f.agenda.Book(f.gym.ID, a, f.session.ID)
	if err != nil || first.Status != domain.BookingConfirmed {
		t.Fatalf("primeira reserva deveria confirmar: %v %v", first, err)
	}
	second, err := f.agenda.Book(f.gym.ID, b, f.session.ID)
	if err != nil || second.Status != domain.BookingWaitlist {
		t.Fatalf("segunda reserva deveria ir para espera: %v %v", second, err)
	}
	if _, err := f.agenda.Book(f.gym.ID, a, f.session.ID); err == nil {
		t.Fatal("reserva duplicada deveria falhar")
	}
}

func TestCancelPromotesWaitlist(t *testing.T) {
	f := setup(t, 1, 24*time.Hour)
	a, b := f.student(t, nil), f.student(t, nil)
	f.agenda.Book(f.gym.ID, a, f.session.ID)
	f.agenda.Book(f.gym.ID, b, f.session.ID)

	if err := f.agenda.Cancel(f.gym.ID, a, f.session.ID); err != nil {
		t.Fatalf("cancelamento falhou: %v", err)
	}
	var bk domain.Booking
	f.db.Where("student_id = ?", b).First(&bk)
	if bk.Status != domain.BookingConfirmed {
		t.Fatalf("lista de espera deveria ser promovida, status=%s", bk.Status)
	}
	// Reservar de novo após cancelar volta para o fim da fila
	again, err := f.agenda.Book(f.gym.ID, a, f.session.ID)
	if err != nil || again.Status != domain.BookingWaitlist {
		t.Fatalf("nova reserva deveria entrar na espera: %v %v", again, err)
	}
}

func TestCancelDeadline(t *testing.T) {
	f := setup(t, 5, time.Hour) // começa em 1h, prazo é 2h antes
	a := f.student(t, nil)
	if _, err := f.agenda.Book(f.gym.ID, a, f.session.ID); err != nil {
		t.Fatal(err)
	}
	err := f.agenda.Cancel(f.gym.ID, a, f.session.ID)
	if err == nil || !IsBusiness(err) {
		t.Fatalf("cancelamento fora do prazo deveria falhar com erro de negócio, veio %v", err)
	}
}

func TestBookingWindowAndInactive(t *testing.T) {
	f := setup(t, 5, 10*24*time.Hour) // além da janela de 7 dias
	a := f.student(t, nil)
	if _, err := f.agenda.Book(f.gym.ID, a, f.session.ID); err == nil {
		t.Fatal("reserva fora da janela deveria falhar")
	}

	f2 := setup(t, 5, 24*time.Hour)
	b := f2.student(t, nil)
	f2.db.Model(&domain.Student{}).Where("id = ?", b).Update("status", domain.StatusInadimplente)
	if _, err := f2.agenda.Book(f2.gym.ID, b, f2.session.ID); err == nil {
		t.Fatal("aluna inativa não deveria reservar")
	}
}

func TestWeeklyPlanLimit(t *testing.T) {
	f := setup(t, 5, 24*time.Hour)
	plan := domain.Plan{GymID: f.gym.ID, Name: "1x", ClassesPerWeek: 1, Active: true}
	f.db.Create(&plan)
	a := f.student(t, &plan)

	if _, err := f.agenda.Book(f.gym.ID, a, f.session.ID); err != nil {
		t.Fatal(err)
	}
	// Outra aula na mesma semana da primeira
	other := f.session
	other.ID = uuid.Nil
	other.StartsAt = f.session.StartsAt.Add(time.Hour)
	other.EndsAt = other.StartsAt.Add(time.Hour)
	f.db.Create(&other)
	if _, w1 := weekBounds(f.session.StartsAt); !other.StartsAt.Before(w1) {
		t.Skip("aulas caíram em semanas diferentes")
	}
	if _, err := f.agenda.Book(f.gym.ID, a, other.ID); err == nil {
		t.Fatal("limite semanal do plano deveria bloquear")
	}
}

func TestTrialBooking(t *testing.T) {
	f := setup(t, 1, 24*time.Hour)
	lead := domain.Lead{GymID: f.gym.ID, Name: "Lead"}
	f.db.Create(&lead)
	if _, err := f.agenda.BookTrial(f.db, f.gym.ID, lead.ID, f.session.ID); err != nil {
		t.Fatalf("aula experimental deveria reservar: %v", err)
	}
	lead2 := domain.Lead{GymID: f.gym.ID, Name: "Lead 2"}
	f.db.Create(&lead2)
	if _, err := f.agenda.BookTrial(f.db, f.gym.ID, lead2.ID, f.session.ID); err == nil {
		t.Fatal("aula lotada não deveria aceitar experimental")
	}
}

func TestCheckinCodes(t *testing.T) {
	c := NewCheckinCodes([]byte("segredo"))
	gym, other := uuid.New(), uuid.New()
	now := time.Now()
	code, _ := c.Current(gym, now)

	if !c.Valid(gym, code, now) {
		t.Fatal("código atual deveria valer")
	}
	if !c.Valid(gym, code, now.Add(45*time.Second)) {
		t.Fatal("código deveria valer dentro da tolerância")
	}
	if c.Valid(gym, code, now.Add(3*time.Minute)) {
		t.Fatal("código antigo não deveria valer")
	}
	if c.Valid(other, code, now) {
		t.Fatal("código de outra academia não deveria valer")
	}
}
