package repository

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/pkg/mockbilling"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedAdminDemo dados de demonstração da gestão: equipe, histórico de alunos, mensalidades, vendas, aulas passadas
func seedAdminDemo(db *gorm.DB, gym domain.Gym, hash string) {
	gymID := gym.ID
	rng := rand.New(rand.NewSource(42))
	loc := config.Loc()
	now := config.Now()

	staff := func(name, email string, role domain.Role, title string, p domain.StaffProfile) domain.User {
		u := domain.User{GymID: gymID, Name: name, Email: email, PasswordHash: hash, Role: role, JobTitle: title, Active: true, FirstAccess: false}
		db.Where(domain.User{Email: email}).FirstOrCreate(&u)
		if u.JobTitle == "" {
			db.Model(&u).Update("job_title", title)
		}
		var n int64
		db.Model(&domain.StaffProfile{}).Where("user_id = ?", u.ID).Count(&n)
		if n == 0 {
			p.GymID, p.UserID = gymID, u.ID
			db.Create(&p)
		}
		return u
	}
	hire := now.AddDate(-1, -2, 0)
	owner := staff("Gestão Unika", "admin@unika.app", domain.RoleAdmin, "Proprietária", domain.StaffProfile{PayType: domain.PayFixed})
	gerente := staff("Marina Gerente", "gerente@unika.app", domain.RoleGerente, "Gerente da unidade", domain.StaffProfile{PayType: domain.PayFixed, BaseSalary: 4200, CommissionPercent: 2, HireDate: &hire, PixKey: "gerente@unika.app"})
	recepcao := staff("Júlia Recepção", "recepcao@unika.app", domain.RoleRecepcao, "Recepcionista", domain.StaffProfile{PayType: domain.PayFixed, BaseSalary: 2100, CommissionFixed: 15, HireDate: &hire, PixKey: "11988887777"})
	consultora := staff("Bruna Consultora", "consultora@unika.app", domain.RoleConsultor, "Consultora de vendas", domain.StaffProfile{PayType: domain.PayFixed, BaseSalary: 1800, CommissionPercent: 8, HireDate: &hire, PixKey: "consultora@unika.app"})
	prof := staff("Camila Professora", "professora@unika.app", domain.RoleProfessor, "Professora de Pilates", domain.StaffProfile{PayType: domain.PayMixed, BaseSalary: 1200, HourlyRate: 45, HireDate: &hire, PixKey: "professora@unika.app"})
	prof2 := staff("Rafaela Personal", "rafaela@unika.app", domain.RoleProfessor, "Professora de Funcional", domain.StaffProfile{PayType: domain.PayHourly, HourlyRate: 55, HireDate: &hire})
	sellers := []domain.User{consultora, consultora, consultora, recepcao, gerente, owner}

	// Reparte a grade entre as duas professoras (Funcional e Dança com a Rafaela)
	var mods []domain.Modality
	db.Where("gym_id = ?", gymID).Find(&mods)
	for _, m := range mods {
		if m.Name == "Funcional" || m.Name == "Dança Fit" {
			db.Model(&domain.ClassSchedule{}).Where("gym_id = ? AND modality_id = ? AND teacher_id = ?", gymID, m.ID, prof.ID).Update("teacher_id", prof2.ID)
			db.Model(&domain.ClassSession{}).Where("gym_id = ? AND modality_id = ? AND teacher_id = ?", gymID, m.ID, prof.ID).Update("teacher_id", prof2.ID)
		}
	}

	var nStudents int64
	db.Model(&domain.Student{}).Where("gym_id = ?", gymID).Count(&nStudents)
	if nStudents >= 10 {
		return // histórico já gerado
	}
	log.Println("[SEED] Gerando histórico de demonstração da gestão...")

	var plans []domain.Plan
	db.Where("gym_id = ?", gymID).Order("price asc").Find(&plans)
	if len(plans) == 0 {
		return
	}
	names := []string{
		"Beatriz Souza", "Larissa Oliveira", "Fernanda Lima", "Patrícia Rocha", "Aline Martins", "Juliana Alves",
		"Carolina Ribeiro", "Gabriela Costa", "Renata Carvalho", "Vanessa Gomes", "Tatiane Freitas", "Priscila Mendes",
		"Letícia Barbosa", "Amanda Teixeira", "Débora Pinto", "Natália Araújo", "Isabela Moreira", "Mariana Castro",
		"Camila Duarte", "Bianca Nunes", "Luana Cardoso", "Sabrina Ramos", "Paula Monteiro", "Elaine Correia",
	}
	cancelReasons := []string{"Mudou de cidade", "Questão financeira", "Horários não encaixam", "Lesão / recomendação médica"}
	methods := []domain.PaymentMethod{domain.PayPix, domain.PayPix, domain.PayCard, domain.PayCard, domain.PayCash, domain.PayBoleto}
	gw := mockbilling.MockGateway{}

	var profiles []domain.StaffProfile
	db.Where("gym_id = ?", gymID).Find(&profiles)
	profileBy := map[uuid.UUID]*domain.StaffProfile{}
	for i := range profiles {
		profileBy[profiles[i].UserID] = &profiles[i]
	}

	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	var studentIDs []uuid.UUID
	for i, name := range names {
		email := fmt.Sprintf("demo.aluna%02d@unika.app", i+1)
		created := now.AddDate(0, 0, -rng.Intn(200)-3)
		u := domain.User{GymID: gymID, Name: name, Email: email, Phone: fmt.Sprintf("1199%07d", rng.Intn(9999999)), PasswordHash: hash,
			Role: domain.RoleAluno, Active: true, FirstAccess: false, CreatedAt: created}
		db.Create(&u)
		plan := plans[rng.Intn(len(plans))]
		st := domain.Student{GymID: gymID, UserID: u.ID, PlanID: &plan.ID, PlanName: plan.Name, PlanPrice: plan.Price,
			DueDay: []int{5, 10, 15, 20}[rng.Intn(4)], Status: domain.StatusAtivo, CreatedAt: created, ImportSource: "demo"}
		canceled := i%8 == 7
		if canceled {
			at := created.AddDate(0, 0, 30+rng.Intn(60))
			if at.After(now) {
				at = now.AddDate(0, 0, -2)
			}
			st.Status, st.CanceledAt, st.CancelReason = domain.StatusCancelado, &at, cancelReasons[rng.Intn(len(cancelReasons))]
		}
		db.Create(&st)
		studentIDs = append(studentIDs, st.ID)

		// Venda da matrícula
		seller := sellers[rng.Intn(len(sellers))]
		amount := plan.Price * float64(max(plan.DurationMonths, 1))
		comm := 0.0
		if p := profileBy[seller.ID]; p != nil {
			comm = amount*p.CommissionPercent/100 + p.CommissionFixed
		}
		sid := st.ID
		db.Create(&domain.Sale{GymID: gymID, SellerID: seller.ID, StudentID: &sid, PlanID: &plan.ID, Kind: domain.SaleNew,
			Description: plan.Name, Amount: amount, Commission: comm, Status: domain.SaleActive, CreatedByID: seller.ID, CreatedAt: created})

		// Mensalidades: da matrícula até o mês atual
		end := now
		if st.CanceledAt != nil {
			end = *st.CanceledAt
		}
		for m := time.Date(created.Year(), created.Month(), 1, 0, 0, 0, 0, loc); !m.After(end); m = m.AddDate(0, 1, 0) {
			due := time.Date(m.Year(), m.Month(), st.DueDay, 0, 0, 0, 0, loc)
			if m.Year() == created.Year() && m.Month() == created.Month() {
				due = time.Date(created.Year(), created.Month(), created.Day(), 0, 0, 0, 0, loc)
			}
			inv := domain.Invoice{GymID: gymID, StudentID: st.ID, Kind: domain.InvoiceMonthly, Reference: m.Format("2006-01"),
				Amount: plan.Price, DueDate: due, Description: fmt.Sprintf("Mensalidade %s — %s", m.Format("01/2006"), plan.Name), CreatedAt: due.AddDate(0, 0, -10)}
			gw.Issue(&inv, &gym)
			switch {
			case due.After(now):
				inv.Status = domain.InvoicePendente
			case m.Equal(thisMonth) && rng.Intn(4) == 0, m.Equal(thisMonth.AddDate(0, -1, 0)) && rng.Intn(7) == 0:
				inv.Status = domain.InvoiceVencido
			default:
				paidAt := due.AddDate(0, 0, rng.Intn(6)-3).Add(time.Duration(9+rng.Intn(10)) * time.Hour)
				if paidAt.After(time.Now()) {
					paidAt = time.Now().Add(-time.Hour)
				}
				inv.Status, inv.PaidAt, inv.PaidAmount = domain.InvoicePago, &paidAt, plan.Price
				inv.PaymentMethod = methods[rng.Intn(len(methods))]
				if rng.Intn(10) == 0 {
					inv.Discount = 10
					inv.PaidAmount = plan.Price - 10
				}
			}
			db.Create(&inv)
		}
	}

	// Renovações recentes
	for i := 0; i < 4; i++ {
		sid := studentIDs[rng.Intn(len(studentIDs))]
		plan := plans[len(plans)-1]
		amount := plan.Price * float64(max(plan.DurationMonths, 1))
		comm := amount * profileBy[consultora.ID].CommissionPercent / 100
		db.Create(&domain.Sale{GymID: gymID, SellerID: consultora.ID, StudentID: &sid, PlanID: &plan.ID, Kind: domain.SaleRenewal,
			Description: plan.Name + " (renovação)", Amount: amount, Commission: comm, Status: domain.SaleActive, CreatedByID: consultora.ID,
			CreatedAt: now.AddDate(0, 0, -rng.Intn(now.Day()))})
	}

	// Metas do mês
	ref := thisMonth.Format("2006-01")
	db.Create(&domain.Goal{GymID: gymID, Month: ref, TargetSales: 15, TargetRevenue: 6000})
	db.Create(&domain.Goal{GymID: gymID, UserID: &consultora.ID, Month: ref, TargetSales: 10, TargetRevenue: 4000})
	db.Create(&domain.Goal{GymID: gymID, UserID: &recepcao.ID, Month: ref, TargetSales: 3, TargetRevenue: 900})

	// Leads com responsáveis
	leadNames := []string{"Sofia Andrade", "Helena Prado", "Valentina Reis", "Manuela Faria", "Alice Moura", "Laura Batista",
		"Lívia Campos", "Cecília Lopes", "Eloá Vieira", "Clara Fonseca", "Yasmin Rezende", "Lorena Dias"}
	statuses := []domain.LeadStatus{domain.LeadNew, domain.LeadNew, domain.LeadScheduled, domain.LeadAttended, domain.LeadEnrolled, domain.LeadLost}
	sources := []string{"site", "instagram", "indicacao", "recepcao"}
	for i, n := range leadNames {
		l := domain.Lead{GymID: gymID, Name: n, Phone: fmt.Sprintf("1198%07d", rng.Intn(9999999)), Source: sources[rng.Intn(len(sources))],
			Status: statuses[rng.Intn(len(statuses))], CreatedAt: now.AddDate(0, 0, -rng.Intn(25))}
		if i%3 != 2 {
			a := consultora.ID
			if i%4 == 0 {
				a = recepcao.ID
			}
			l.AssignedToID = &a
		}
		db.Create(&l)
	}

	// Aulas dos últimos 30 dias com presença, faltas e check-ins (alimenta relatórios e horas/aula)
	var schedules []domain.ClassSchedule
	db.Where("gym_id = ? AND active = ?", gymID, true).Find(&schedules)
	active := studentIDs[:len(studentIDs)-3]
	for d := 30; d >= 1; d-- {
		day := config.StartOfDay(now).AddDate(0, 0, -d)
		for _, sc := range schedules {
			if int(day.Weekday()) != sc.Weekday {
				continue
			}
			var hh, mm int
			fmt.Sscanf(sc.StartTime, "%d:%d", &hh, &mm)
			start := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, loc)
			scID := sc.ID
			teacher := sc.TeacherID
			if rng.Intn(15) == 0 { // de vez em quando alguém cobre a aula
				if teacher != nil && *teacher == prof.ID {
					teacher = &prof2.ID
				} else {
					teacher = &prof.ID
				}
			}
			s := domain.ClassSession{GymID: gymID, ScheduleID: &scID, ModalityID: sc.ModalityID, TeacherID: teacher, StartsAt: start.UTC(),
				EndsAt: start.Add(time.Duration(sc.DurationMin) * time.Minute).UTC(), Capacity: sc.Capacity, Room: sc.Room, Status: domain.SessionScheduled}
			if db.Create(&s).Error != nil {
				continue
			}
			n := rng.Intn(sc.Capacity + 1)
			perm := rng.Perm(len(active))
			for k := 0; k < n && k < len(perm); k++ {
				stID := active[perm[k]]
				status := domain.BookingAttended
				if rng.Intn(6) == 0 {
					status = domain.BookingNoShow
				}
				b := domain.Booking{GymID: gymID, SessionID: s.ID, StudentID: &stID, Status: status, CreatedAt: start.AddDate(0, 0, -1)}
				db.Create(&b)
				if status == domain.BookingAttended {
					bid := b.ID
					db.Create(&domain.CheckIn{GymID: gymID, StudentID: stID, BookingID: &bid, Method: "QR", CreatedAt: start.Add(-time.Duration(rng.Intn(15)) * time.Minute)})
				}
			}
		}
	}

	// Equipamentos
	last := now.AddDate(0, -1, 0)
	next := now.AddDate(0, 0, 12)
	db.Create(&[]domain.Equipment{
		{GymID: gymID, Name: "Crossover Polia Dupla", Sector: "Musculação", Status: domain.EquipOK, Notes: "Cabos trocados", LastMaintenance: &last, NextMaintenance: &next},
		{GymID: gymID, Name: "Esteira #03", Sector: "Cardio", Status: domain.EquipMaintenance, Notes: "Troca de lona agendada"},
		{GymID: gymID, Name: "Leg Press 45°", Sector: "Musculação", Status: domain.EquipOK, LastMaintenance: &last},
		{GymID: gymID, Name: "Reformer de Pilates #2", Sector: "Pilates", Status: domain.EquipWarning, Notes: "Mola com ruído"},
		{GymID: gymID, Name: "Bicicleta Ergométrica #1", Sector: "Cardio", Status: domain.EquipOK, NextMaintenance: &next},
	})

	// Algumas entradas no histórico de ações
	logs := []domain.AuditLog{
		{GymID: gymID, UserID: &recepcao.ID, UserName: recepcao.Name, UserRole: recepcao.Role, Action: "invoice.pay", Entity: "invoice", Summary: "Recebeu R$ 149,90 (DINHEIRO) — Beatriz Souza", CreatedAt: now.Add(-3 * time.Hour)},
		{GymID: gymID, UserID: &consultora.ID, UserName: consultora.Name, UserRole: consultora.Role, Action: "lead.convert", Entity: "student", Summary: "Matriculou o lead Helena Prado (Trimestral Livre, R$ 169,90)", CreatedAt: now.Add(-26 * time.Hour)},
		{GymID: gymID, UserID: &gerente.ID, UserName: gerente.Name, UserRole: gerente.Role, Action: "session.substitute", Entity: "session", Summary: "Substituição na Funcional: Rafaela Personal → Camila Professora", CreatedAt: now.Add(-50 * time.Hour)},
	}
	db.Create(&logs)
	log.Println("[SEED] Histórico da gestão pronto: gerente@ / recepcao@ / consultora@ / rafaela@unika.app (mesma senha)")
}
