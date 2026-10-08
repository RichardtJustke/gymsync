package service

import (
	"testing"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
)

func (f *billFixture) staff(t *testing.T, role domain.Role, profile *domain.StaffProfile) domain.User {
	t.Helper()
	u := domain.User{GymID: f.gym.ID, Name: string(role), Email: uuid.NewString() + "@x.com", PasswordHash: "x", Role: role, Active: true}
	f.db.Create(&u)
	if profile != nil {
		profile.GymID, profile.UserID = f.gym.ID, u.ID
		f.db.Create(profile)
	}
	return u
}

func TestCommissionFor(t *testing.T) {
	p := &domain.StaffProfile{CommissionPercent: 10, CommissionFixed: 5}
	if c := CommissionFor(p, 450); c != 50 {
		t.Fatalf("10%% de 450 + 5 = 50, veio %v", c)
	}
	if CommissionFor(nil, 450) != 0 {
		t.Fatal("sem perfil não há comissão")
	}
}

func TestRegisterSaleUsesContractValueAndFreezesCommission(t *testing.T) {
	f := setupBilling(t)
	seller := f.staff(t, domain.RoleConsultor, &domain.StaffProfile{CommissionPercent: 10})
	plan := domain.Plan{GymID: f.gym.ID, Name: "Trimestral", Price: 150, DurationMonths: 3, Active: true}
	f.db.Create(&plan)

	sale, err := f.team.RegisterSale(f.db, f.gym.ID, SaleInput{SellerID: seller.ID, PlanID: &plan.ID, Kind: domain.SaleNew})
	if err != nil {
		t.Fatal(err)
	}
	if sale.Amount != 450 || sale.Commission != 45 {
		t.Fatalf("contrato 3x150 = 450 e comissão 45, veio %v / %v", sale.Amount, sale.Commission)
	}
	// Mudar a regra depois não altera a venda já feita
	f.db.Model(&domain.StaffProfile{}).Where("user_id = ?", seller.ID).Update("commission_percent", 50)
	var stored domain.Sale
	f.db.First(&stored, "id = ?", sale.ID)
	if stored.Commission != 45 {
		t.Fatalf("comissão gravada deveria continuar 45, veio %v", stored.Commission)
	}

	if _, err := f.team.RegisterSale(f.db, f.gym.ID, SaleInput{SellerID: uuid.New(), Kind: domain.SaleNew, Amount: 10}); err == nil {
		t.Fatal("vendedor inexistente deveria falhar")
	}
	inactive := f.staff(t, domain.RoleConsultor, nil)
	f.db.Model(&inactive).Update("active", false)
	if _, err := f.team.RegisterSale(f.db, f.gym.ID, SaleInput{SellerID: inactive.ID, Kind: domain.SaleNew, Amount: 10}); err == nil {
		t.Fatal("vendedor desativado deveria falhar")
	}
}

func TestRankingWithGoals(t *testing.T) {
	f := setupBilling(t)
	a := f.staff(t, domain.RoleConsultor, &domain.StaffProfile{CommissionFixed: 20})
	b := f.staff(t, domain.RoleConsultor, nil)
	month, _ := ParseMonth("")
	ref := month.Format("2006-01")
	f.db.Create(&domain.Goal{GymID: f.gym.ID, UserID: &a.ID, Month: ref, TargetRevenue: 1000})
	f.db.Create(&domain.Goal{GymID: f.gym.ID, Month: ref, TargetSales: 4})
	f.team.RegisterSale(f.db, f.gym.ID, SaleInput{SellerID: a.ID, Kind: domain.SaleNew, Amount: 500})
	f.team.RegisterSale(f.db, f.gym.ID, SaleInput{SellerID: a.ID, Kind: domain.SaleRenewal, Amount: 250})
	f.team.RegisterSale(f.db, f.gym.ID, SaleInput{SellerID: b.ID, Kind: domain.SaleNew, Amount: 100})

	r := f.team.Ranking(f.gym.ID, month)
	if len(r.Rows) != 2 || r.Rows[0].UserID != a.ID {
		t.Fatalf("ranking deveria ter A em primeiro: %+v", r.Rows)
	}
	if r.Rows[0].Progress != 75 || r.Rows[0].Commission != 40 || r.Rows[0].Renewals != 1 {
		t.Fatalf("linha de A errada: %+v", r.Rows[0])
	}
	if r.GymProgress != 75 || r.TotalSales != 3 {
		t.Fatalf("meta da academia: 3 de 4 vendas = 75%%, veio %v (%d vendas)", r.GymProgress, r.TotalSales)
	}
}

func TestPayrollMixedAndPaidSnapshot(t *testing.T) {
	f := setupBilling(t)
	month, _ := ParseMonth("")
	prof := f.staff(t, domain.RoleProfessor, &domain.StaffProfile{PayType: domain.PayMixed, BaseSalary: 1000, HourlyRate: 50})
	mod := domain.Modality{GymID: f.gym.ID, Name: "Pilates", Active: true}
	f.db.Create(&mod)
	// Uma aula de 1h30 já dada neste mês (no início do mês, para não depender do dia de hoje)
	start := config.Now().Add(-3 * time.Hour)
	if start.Before(month) {
		start = month.Add(time.Hour)
	}
	if start.Add(90 * time.Minute).After(time.Now()) {
		t.Skip("início do mês: sem janela para uma aula já encerrada")
	}
	f.db.Create(&domain.ClassSession{GymID: f.gym.ID, ModalityID: mod.ID, TeacherID: &prof.ID, StartsAt: start, EndsAt: start.Add(90 * time.Minute), Capacity: 10, Status: domain.SessionScheduled})

	row, err := f.team.SavePayroll(f.gym.ID, prof.ID, month, PayrollAdjust{Bonus: 100, Deductions: 25})
	if err != nil {
		t.Fatal(err)
	}
	if row.Hours != 1.5 || row.HoursAmount != 75 || row.Total != 1150 {
		t.Fatalf("1000 + 1,5h×50 + 100 − 25 = 1150, veio %+v", row)
	}
	row, err = f.team.SavePayroll(f.gym.ID, prof.ID, month, PayrollAdjust{Bonus: 100, Deductions: 25, MarkPaid: true})
	if err != nil || row.Status != domain.PayrollPaid {
		t.Fatalf("deveria marcar como pago: %+v %v", row, err)
	}
	// Depois de pago, mudar o salário não altera o fechamento
	f.db.Model(&domain.StaffProfile{}).Where("user_id = ?", prof.ID).Update("base_salary", 5000)
	for _, r := range f.team.Payroll(f.gym.ID, month) {
		if r.UserID == prof.ID && r.Total != 1150 {
			t.Fatalf("fechamento pago deveria estar congelado em 1150, veio %v", r.Total)
		}
	}
	if _, err := f.team.SavePayroll(f.gym.ID, prof.ID, month, PayrollAdjust{Bonus: 1}); err == nil {
		t.Fatal("alterar fechamento pago sem reabrir deveria falhar")
	}
}
