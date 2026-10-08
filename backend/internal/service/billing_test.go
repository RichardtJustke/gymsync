package service

import (
	"testing"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeGateway struct{ issued int }

func (g *fakeGateway) Name() string { return "fake" }
func (g *fakeGateway) Issue(inv *domain.Invoice, gym *domain.Gym) error {
	g.issued++
	inv.BoletoDigits = "000"
	return nil
}

type billFixture struct {
	db      *gorm.DB
	billing *BillingService
	team    *TeamService
	gw      *fakeGateway
	gym     domain.Gym
}

func setupBilling(t *testing.T) *billFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Gym{}, &domain.User{}, &domain.Student{}, &domain.Plan{}, &domain.Invoice{},
		&domain.Notification{}, &domain.DeviceToken{}, &domain.StaffProfile{}, &domain.Sale{}, &domain.Goal{},
		&domain.Lead{}, &domain.PayrollEntry{}, &domain.ClassSchedule{}, &domain.ClassSession{}, &domain.Booking{}, &domain.Modality{}); err != nil {
		t.Fatal(err)
	}
	gym := domain.Gym{Name: "Teste", Document: "1"}
	db.Create(&gym)
	gw := &fakeGateway{}
	notifier := NewNotifier(db, LogPushSender{})
	return &billFixture{db: db, billing: NewBillingService(db, gw, notifier), team: NewTeamService(db), gw: gw, gym: gym}
}

func (f *billFixture) student(t *testing.T, price float64, dueDay int, created time.Time) domain.Student {
	t.Helper()
	u := domain.User{GymID: f.gym.ID, Name: "Aluna", Email: uuid.NewString() + "@x.com", PasswordHash: "x", Role: domain.RoleAluno}
	f.db.Create(&u)
	st := domain.Student{GymID: f.gym.ID, UserID: u.ID, Status: domain.StatusAtivo, PlanName: "Mensal", PlanPrice: price, DueDay: dueDay, CreatedAt: created}
	f.db.Create(&st)
	return st
}

func TestDueDateForClampsShortMonths(t *testing.T) {
	feb := time.Date(2026, 2, 1, 0, 0, 0, 0, config.Loc())
	if d := DueDateFor(feb, 31); d.Day() != 28 {
		t.Fatalf("dia 31 em fevereiro deveria virar 28, veio %d", d.Day())
	}
	if d := DueDateFor(feb, 0); d.Day() != 10 {
		t.Fatalf("dia inválido deveria virar 10, veio %d", d.Day())
	}
}

func TestGenerateMonthIsIdempotentAndSkipsFreeAndLate(t *testing.T) {
	f := setupBilling(t)
	month := time.Date(2026, 3, 1, 0, 0, 0, 0, config.Loc())
	old := month.AddDate(0, -2, 0)
	f.student(t, 150, 10, old)                    // cobra
	f.student(t, 0, 10, old)                      // plano gratuito: não cobra
	f.student(t, 150, 5, month.AddDate(0, 0, 20)) // entrou depois do vencimento: pagou na matrícula
	n, err := f.billing.GenerateMonth(f.gym.ID, month)
	if err != nil || n != 1 {
		t.Fatalf("esperava 1 cobrança, veio %d (%v)", n, err)
	}
	n, _ = f.billing.GenerateMonth(f.gym.ID, month)
	if n != 0 {
		t.Fatalf("segunda geração não deveria duplicar, criou %d", n)
	}
}

func TestOverdueBlocksAndPaymentUnblocks(t *testing.T) {
	f := setupBilling(t)
	f.db.Model(&f.gym).Update("overdue_block_days", 3)
	st := f.student(t, 100, 1, time.Now().AddDate(0, -3, 0))
	inv := domain.Invoice{GymID: f.gym.ID, StudentID: st.ID, Kind: domain.InvoiceMonthly, Reference: "2020-01", Amount: 100,
		DueDate: config.StartOfDay(config.Now()).AddDate(0, 0, -10)}
	if err := f.billing.Create(f.db, &inv); err != nil {
		t.Fatal(err)
	}
	overdue, blocked := f.billing.RefreshOverdue(f.gym.ID)
	if overdue != 1 || blocked != 1 {
		t.Fatalf("esperava 1 vencida e 1 bloqueada, veio %d/%d", overdue, blocked)
	}
	var s domain.Student
	f.db.First(&s, "id = ?", st.ID)
	if s.Status != domain.StatusInadimplente {
		t.Fatalf("aluna deveria estar inadimplente, está %s", s.Status)
	}

	paid, err := f.billing.RegisterPayment(f.gym.ID, inv.ID, PaymentInput{Method: domain.PayCash, Discount: 10})
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaidAmount != 90 || paid.Status != domain.InvoicePago {
		t.Fatalf("baixa com desconto errada: %+v", paid)
	}
	f.db.First(&s, "id = ?", st.ID)
	if s.Status != domain.StatusAtivo {
		t.Fatalf("pagamento deveria liberar a aluna, está %s", s.Status)
	}
	if _, err := f.billing.RegisterPayment(f.gym.ID, inv.ID, PaymentInput{Method: domain.PayCash}); err == nil {
		t.Fatal("pagar duas vezes deveria falhar")
	}
	if _, err := f.billing.Cancel(f.gym.ID, inv.ID, "x"); err == nil {
		t.Fatal("cancelar cobrança paga deveria falhar")
	}
}

func TestOverdueWithoutBlockKeepsStudentActive(t *testing.T) {
	f := setupBilling(t)
	st := f.student(t, 100, 1, time.Now().AddDate(0, -3, 0))
	inv := domain.Invoice{GymID: f.gym.ID, StudentID: st.ID, Amount: 100, DueDate: config.StartOfDay(config.Now()).AddDate(0, 0, -30)}
	f.billing.Create(f.db, &inv)
	_, blocked := f.billing.RefreshOverdue(f.gym.ID)
	if blocked != 0 {
		t.Fatalf("sem regra de bloqueio ninguém deveria ser bloqueado, bloqueou %d", blocked)
	}
}

func TestSummary(t *testing.T) {
	f := setupBilling(t)
	month, _ := ParseMonth("")
	st := f.student(t, 100, 1, time.Now().AddDate(0, -3, 0))
	a := domain.Invoice{GymID: f.gym.ID, StudentID: st.ID, Amount: 100, DueDate: month.AddDate(0, 0, 1)}
	b := domain.Invoice{GymID: f.gym.ID, StudentID: st.ID, Amount: 50, DueDate: month.AddDate(0, 0, 2)}
	f.billing.Create(f.db, &a)
	f.billing.Create(f.db, &b)
	f.billing.RegisterPayment(f.gym.ID, a.ID, PaymentInput{Method: domain.PayPix})
	s := f.billing.Summary(f.gym.ID, month)
	if s.Expected != 150 || s.Received != 100 || s.Pending != 50 || s.ByMethod["PIX"] != 100 {
		t.Fatalf("resumo errado: %+v", s)
	}
}

func TestFormatBRL(t *testing.T) {
	cases := map[float64]string{0: "0,00", 1234.5: "1.234,50", 1000000: "1.000.000,00", 99.999: "100,00"}
	for in, want := range cases {
		if got := FormatBRL(in); got != want {
			t.Errorf("FormatBRL(%v) = %s, esperava %s", in, got, want)
		}
	}
}
