package repository

import (
	"log"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Senha das contas de demonstração (somente desenvolvimento, SEED_DEMO=true)
const demoPassword = "unika2026"

var demoGymID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// SeedDatabase popula a base global de exercícios e, em desenvolvimento, dados de demonstração da Unika
func SeedDatabase(db *gorm.DB) error {
	var count int64
	db.Model(&domain.Exercise{}).Where("gym_id IS NULL").Count(&count)
	if count == 0 {
		log.Println("[SEED] Inserindo base global de exercícios...")
		if err := db.CreateInBatches(getComprehensiveExercises(), 50).Error; err != nil {
			return err
		}
	}

	var gyms int64
	db.Model(&domain.Gym{}).Count(&gyms)
	if gyms == 0 {
		gym := domain.Gym{ID: demoGymID, Name: "Unika Fitness", Document: "00.000.000/0001-00", ThemeHex: "#FF5A1F", BookingWindowDays: 7, CancelDeadlineHours: 2}
		if err := db.Create(&gym).Error; err != nil {
			return err
		}
		log.Println("[SEED] Academia criada: Unika Fitness")
	}

	if err := seedWorkoutTemplates(db); err != nil {
		log.Printf("[AVISO] Falha ao criar modelos de treino padrão: %v", err)
	}

	if !config.C.SeedDemo {
		return nil
	}
	return seedDemo(db)
}

func seedDemo(db *gorm.DB) error {
	var gym domain.Gym
	if err := db.Order("created_at asc").First(&gym).Error; err != nil {
		return err
	}
	gymID := gym.ID
	hash, _ := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)

	user := func(name, email, doc string, role domain.Role) domain.User {
		u := domain.User{GymID: gymID, Name: name, Email: email, Document: doc, PasswordHash: string(hash), Role: role, FirstAccess: false}
		db.Where(domain.User{Email: email}).FirstOrCreate(&u)
		return u
	}
	user("Gestão Unika", "admin@unika.app", "", domain.RoleAdmin)
	prof := user("Camila Professora", "professora@unika.app", "", domain.RoleProfessor)
	aluna := user("Ana Aluna", "aluna@unika.app", "", domain.RoleAluno)

	// Planos
	var nPlans int64
	db.Model(&domain.Plan{}).Where("gym_id = ?", gymID).Count(&nPlans)
	if nPlans == 0 {
		db.Create(&[]domain.Plan{
			{GymID: gymID, Name: "Mensal Livre", Price: 189.90, DurationMonths: 1, ClassesPerWeek: 0, Active: true, Description: "Aulas ilimitadas"},
			{GymID: gymID, Name: "Mensal 3x", Price: 149.90, DurationMonths: 1, ClassesPerWeek: 3, Active: true, Description: "Até 3 aulas por semana"},
			{GymID: gymID, Name: "Trimestral Livre", Price: 169.90, DurationMonths: 3, ClassesPerWeek: 0, Active: true, Description: "Valor mensal no plano de 3 meses"},
		})
	}
	var plan domain.Plan
	db.Where("gym_id = ?", gymID).Order("price desc").First(&plan)

	// Modalidades e grade semanal
	var nMods int64
	db.Model(&domain.Modality{}).Where("gym_id = ?", gymID).Count(&nMods)
	if nMods == 0 {
		mods := []domain.Modality{
			{GymID: gymID, Name: "Pilates", Color: "#C084FC", AllowTrial: true, Active: true},
			{GymID: gymID, Name: "Funcional", Color: "#FF5A1F", AllowTrial: true, Active: true},
			{GymID: gymID, Name: "Dança Fit", Color: "#F472B6", AllowTrial: true, Active: true},
			{GymID: gymID, Name: "Musculação Orientada", Color: "#D7FF3A", AllowTrial: false, Active: true},
		}
		db.Create(&mods)
		pid := prof.ID
		var schedules []domain.ClassSchedule
		slots := []struct {
			mod   int
			days  []int
			start string
			cap   int
		}{
			{0, []int{1, 3, 5}, "07:00", 10},
			{1, []int{1, 2, 3, 4, 5}, "18:30", 20},
			{2, []int{2, 4}, "19:30", 25},
			{3, []int{1, 2, 3, 4, 5, 6}, "09:00", 15},
			{0, []int{6}, "10:00", 10},
		}
		for _, sl := range slots {
			for _, d := range sl.days {
				schedules = append(schedules, domain.ClassSchedule{
					GymID: gymID, ModalityID: mods[sl.mod].ID, TeacherID: &pid, Weekday: d, StartTime: sl.start,
					DurationMin: 60, Capacity: sl.cap, Room: "Sala 1", Active: true,
				})
			}
		}
		db.Create(&schedules)
	}

	// Perfil da aluna demo + ficha
	var student domain.Student
	if db.Where("user_id = ?", aluna.ID).First(&student).Error != nil {
		student = domain.Student{GymID: gymID, UserID: aluna.ID, PlanID: &plan.ID, PlanName: plan.Name, PlanPrice: plan.Price,
			DueDay: 10, Status: domain.StatusAtivo, Weight: 62, Height: 165}
		db.Create(&student)
	}
	var nWp int64
	db.Model(&domain.WorkoutPlan{}).Where("student_id = ?", student.ID).Count(&nWp)
	if nWp == 0 {
		wp := domain.WorkoutPlan{GymID: gymID, StudentID: student.ID, TeacherID: prof.ID, Title: "Treino AB — Fortalecimento", Goal: "Força e condicionamento", Active: true, StartDate: time.Now()}
		db.Create(&wp)
		pick := func(name string) uuid.UUID {
			var e domain.Exercise
			db.Where("name = ?", name).First(&e)
			return e.ID
		}
		wa := domain.Workout{WorkoutPlanID: wp.ID, Letter: "A", Name: "Inferiores e Glúteos", OrderIndex: 1}
		wb := domain.Workout{WorkoutPlanID: wp.ID, Letter: "B", Name: "Superiores e Core", OrderIndex: 2}
		db.Create(&wa)
		db.Create(&wb)
		items := []domain.WorkoutExercise{
			{WorkoutID: wa.ID, ExerciseID: pick("Agachamento Livre com Barra"), Sets: 4, Reps: "10-12", LoadKg: 20, RestSeconds: 90, OrderIndex: 1},
			{WorkoutID: wa.ID, ExerciseID: pick("Leg Press 45°"), Sets: 4, Reps: "12", LoadKg: 60, RestSeconds: 60, OrderIndex: 2},
			{WorkoutID: wa.ID, ExerciseID: pick("Cadeira Extensora"), Sets: 3, Reps: "15", LoadKg: 25, RestSeconds: 45, OrderIndex: 3},
			{WorkoutID: wb.ID, ExerciseID: pick("Puxada Frontal Aberta"), Sets: 4, Reps: "10-12", LoadKg: 30, RestSeconds: 60, OrderIndex: 1},
			{WorkoutID: wb.ID, ExerciseID: pick("Supino Reto com Halteres"), Sets: 3, Reps: "12", LoadKg: 8, RestSeconds: 60, OrderIndex: 2},
			{WorkoutID: wb.ID, ExerciseID: pick("Prancha Abdominal Isométrica"), Sets: 3, Reps: "30s", LoadKg: 0, RestSeconds: 30, OrderIndex: 3},
		}
		for _, it := range items {
			if it.ExerciseID != uuid.Nil {
				db.Create(&it)
			}
		}
	}

	seedAdminDemo(db, gym, string(hash))
	// GORM troca false por default:true ao criar; as contas de demonstração não devem pedir troca de senha
	db.Model(&domain.User{}).Where("gym_id = ? AND email LIKE ?", gymID, "%@unika.app").Update("first_access", false)
	log.Printf("[SEED] Demo pronta: admin@unika.app / professora@unika.app / aluna@unika.app (senha %s)", demoPassword)
	return nil
}

func getComprehensiveExercises() []domain.Exercise {
	list := []domain.Exercise{
		// === PEITORAL ===
		{Name: "Supino Reto com Barra", MuscleGroup: "Peitoral", Equipment: "Barra e Banco", Description: "Exercício composto base para peitoral maior."},
		{Name: "Supino Reto com Halteres", MuscleGroup: "Peitoral", Equipment: "Halteres", Description: "Permite maior amplitude e equilíbrio unilateral."},
		{Name: "Supino Inclinado com Barra", MuscleGroup: "Peitoral", Equipment: "Barra e Banco Inclinado", Description: "Enfatiza a porção clavicular (superior) do peitoral."},
		{Name: "Supino Inclinado com Halteres", MuscleGroup: "Peitoral", Equipment: "Halteres", Description: "Excelente para desenvolvimento da parte superior do tórax."},
		{Name: "Supino Declinado com Barra", MuscleGroup: "Peitoral", Equipment: "Barra e Banco Declinado", Description: "Foco na porção inferior do peitoral."},
		{Name: "Crucifixo Reto com Halteres", MuscleGroup: "Peitoral", Equipment: "Halteres", Description: "Isolamento peitoral com alongamento na fase excêntrica."},
		{Name: "Crucifixo Inclinado com Halteres", MuscleGroup: "Peitoral", Equipment: "Halteres", Description: "Alongamento e contração da porção superior."},
		{Name: "Crossover Polia Alta", MuscleGroup: "Peitoral", Equipment: "Cabo", Description: "Foco no feixe inferior e medial do peitoral."},
		{Name: "Crossover Polia Média", MuscleGroup: "Peitoral", Equipment: "Cabo", Description: "Tensão constante em todo o movimento."},
		{Name: "Crossover Polia Baixa", MuscleGroup: "Peitoral", Equipment: "Cabo", Description: "Enfatiza a porção superior do peitoral com cabos."},
		{Name: "Peck Deck / Voador", MuscleGroup: "Peitoral", Equipment: "Máquina", Description: "Isolamento seguro e constante de peito."},
		{Name: "Flexão de Braços (Push-up)", MuscleGroup: "Peitoral", Equipment: "Peso Corporal", Description: "Exercício funcional básico e eficaz."},
		{Name: "Mergulho nas Paralelas (Foco Peitoral)", MuscleGroup: "Peitoral", Equipment: "Barras Paralelas", Description: "Inclinar o tronco à frente para atingir o peitoral inferior."},

		// === DORSAL / COSTAS ===
		{Name: "Puxada Frontal Aberta", MuscleGroup: "Dorsal", Equipment: "Polia / Máquina", Description: "Alargamento da dorsal (latíssimo do dorso)."},
		{Name: "Puxada Frontal com Triângulo", MuscleGroup: "Dorsal", Equipment: "Polia / Máquina", Description: "Pegada neutra para espessura e ativação da dorsal."},
		{Name: "Puxada Supinada", MuscleGroup: "Dorsal", Equipment: "Polia / Máquina", Description: "Maior ativação das fibras inferiores e bíceps."},
		{Name: "Remada Baixa no Cabo (Triângulo)", MuscleGroup: "Dorsal", Equipment: "Cabo", Description: "Construção de espessura de costas e romboides."},
		{Name: "Remada Curvada com Barra (Pronada)", MuscleGroup: "Dorsal", Equipment: "Barra", Description: "Exercício clássico composto para costas espessas."},
		{Name: "Remada Curvada com Barra (Supinada)", MuscleGroup: "Dorsal", Equipment: "Barra", Description: "Variação estilo Yates com foco em latíssimo e bíceps."},
		{Name: "Remada Unilateral com Halter (Serrote)", MuscleGroup: "Dorsal", Equipment: "Halter", Description: "Trabalho unilateral de dorsal com apoio no banco."},
		{Name: "Remada Cavalinho / Barra T", MuscleGroup: "Dorsal", Equipment: "Barra T / Máquina", Description: "Alta capacidade de carga e densidade muscular."},
		{Name: "Barra Fixa Pronada (Pull-up)", MuscleGroup: "Dorsal", Equipment: "Barra Fixa", Description: "Excelente construtor de força e largura dorsal."},
		{Name: "Barra Fixa Supinada (Chin-up)", MuscleGroup: "Dorsal", Equipment: "Barra Fixa", Description: "Ativa fortemente dorsal e bíceps."},
		{Name: "Pulldown na Polia com Corda", MuscleGroup: "Dorsal", Equipment: "Cabo", Description: "Isolamento da dorsal sem flexão de cotovelo."},
		{Name: "Levantamento Terra (Deadlift)", MuscleGroup: "Dorsal", Equipment: "Barra", Description: "Exercício de força global para cadeia posterior."},

		// === OMBROS / DELTOIDES ===
		{Name: "Desenvolvimento Militar com Barra", MuscleGroup: "Ombros", Equipment: "Barra", Description: "Construtor base de força e massa para ombros."},
		{Name: "Desenvolvimento Militar com Halteres", MuscleGroup: "Ombros", Equipment: "Halteres", Description: "Trabalho independente para estabilidade e simetria."},
		{Name: "Desenvolvimento na Máquina", MuscleGroup: "Ombros", Equipment: "Máquina", Description: "Movimento guiado para segurança articular."},
		{Name: "Elevação Lateral com Halteres", MuscleGroup: "Ombros", Equipment: "Halteres", Description: "Isolamento da cabeça lateral do deltoide (largura de ombros)."},
		{Name: "Elevação Lateral na Polia", MuscleGroup: "Ombros", Equipment: "Cabo", Description: "Tensão contínua desde a base do movimento."},
		{Name: "Elevação Frontal com Halteres", MuscleGroup: "Ombros", Equipment: "Halteres", Description: "Foco na porção anterior do deltoide."},
		{Name: "Elevação Frontal com Barra", MuscleGroup: "Ombros", Equipment: "Barra", Description: "Variação com pegada pronada."},
		{Name: "Crucifixo Inverso com Halteres", MuscleGroup: "Ombros", Equipment: "Halteres", Description: "Deltoide posterior e melhora postural."},
		{Name: "Crucifixo Inverso no Peck Deck", MuscleGroup: "Ombros", Equipment: "Máquina", Description: "Isolamento mecânico do deltoide posterior."},
		{Name: "Face Pull com Corda", MuscleGroup: "Ombros", Equipment: "Cabo", Description: "Saúde do manguito rotador e deltoide posterior."},
		{Name: "Remada Alta com Barra", MuscleGroup: "Ombros", Equipment: "Barra", Description: "Trabalho de deltoide lateral e trapézio."},

		// === TRAPÉZIO ===
		{Name: "Encolhimento com Barra", MuscleGroup: "Trapézio", Equipment: "Barra", Description: "Cargas altas para hipertrofia do trapézio superior."},
		{Name: "Encolhimento com Halteres", MuscleGroup: "Trapézio", Equipment: "Halteres", Description: "Amplitude natural e conforto articular."},
		{Name: "Encolhimento no Smith", MuscleGroup: "Trapézio", Equipment: "Máquina Smith", Description: "Guiado para sobrecarga controlada."},

		// === BÍCEPS ===
		{Name: "Rosca Direta com Barra W", MuscleGroup: "Bíceps", Equipment: "Barra W", Description: "Conforto nos punhos com excelente torque no bíceps."},
		{Name: "Rosca Direta com Barra Reta", MuscleGroup: "Bíceps", Equipment: "Barra Reta", Description: "Supinação máxima para pico de bíceps."},
		{Name: "Rosca Alternada com Halteres", MuscleGroup: "Bíceps", Equipment: "Halteres", Description: "Trabalho unilateral com rotação do punho."},
		{Name: "Rosca Martelo com Halteres", MuscleGroup: "Bíceps", Equipment: "Halteres", Description: "Foco no braquial e braquiorradial (espessura do braço)."},
		{Name: "Rosca Martelo na Polia com Corda", MuscleGroup: "Bíceps", Equipment: "Cabo", Description: "Tensão constante no braquial."},
		{Name: "Rosca Scott com Barra W", MuscleGroup: "Bíceps", Equipment: "Banco Scott e Barra", Description: "Isolamento completo sem impulso do tronco."},
		{Name: "Rosca Concentrada", MuscleGroup: "Bíceps", Equipment: "Halter", Description: "Isolamento de pico do bíceps sentado."},
		{Name: "Rosca Inclinada no Banco 45°", MuscleGroup: "Bíceps", Equipment: "Halteres", Description: "Grande alongamento da cabeça longa do bíceps."},
		{Name: "Rosca 21", MuscleGroup: "Bíceps", Equipment: "Barra", Description: "Método de intensidade com amplitude parcial e completa."},

		// === TRÍCEPS ===
		{Name: "Tríceps Pulley com Barra Reta", MuscleGroup: "Tríceps", Equipment: "Cabo", Description: "Exercício clássico de extensão de cotovelo."},
		{Name: "Tríceps Pulley com Corda", MuscleGroup: "Tríceps", Equipment: "Cabo", Description: "Abertura no final para contração máxima da cabeça lateral."},
		{Name: "Tríceps Testa com Barra W", MuscleGroup: "Tríceps", Equipment: "Barra W e Banco", Description: "Foco intenso na cabeça longa do tríceps."},
		{Name: "Tríceps Francês com Halter", MuscleGroup: "Tríceps", Equipment: "Halter", Description: "Alongamento profundo com os braços elevados."},
		{Name: "Tríceps Coice com Halter", MuscleGroup: "Tríceps", Equipment: "Halter", Description: "Contração final unilateral com tronco inclinado."},
		{Name: "Tríceps Coice na Polia", MuscleGroup: "Tríceps", Equipment: "Cabo", Description: "Tensão no movimento inteiro."},
		{Name: "Mergulho no Banco (Tríceps Banco)", MuscleGroup: "Tríceps", Equipment: "Banco", Description: "Movimento com peso do próprio corpo."},
		{Name: "Supino Fechado", MuscleGroup: "Tríceps", Equipment: "Barra e Banco", Description: "Composto pesado para espessura e força de tríceps."},

		// === ANTEBRAÇO ===
		{Name: "Rosca Inversa com Barra", MuscleGroup: "Antebraço", Equipment: "Barra", Description: "Braquiorradial e extensores do punho."},
		{Name: "Rosca de Punho (Flexão de Punho)", MuscleGroup: "Antebraço", Equipment: "Barra ou Halter", Description: "Fortalecimento e volume dos flexores do punho."},
		{Name: "Rosca de Punho Invertida", MuscleGroup: "Antebraço", Equipment: "Barra", Description: "Fortalecimento dos extensores do punho."},

		// === QUADRÍCEPS ===
		{Name: "Agachamento Livre com Barra", MuscleGroup: "Quadríceps", Equipment: "Barra e Rack", Description: "O rei dos exercícios de membros inferiores."},
		{Name: "Agachamento Frontal (Front Squat)", MuscleGroup: "Quadríceps", Equipment: "Barra", Description: "Ênfase dominante no quadríceps e postura ereta."},
		{Name: "Agachamento no Smith", MuscleGroup: "Quadríceps", Equipment: "Máquina Smith", Description: "Controle e segurança da trajetória de movimento."},
		{Name: "Leg Press 45°", MuscleGroup: "Quadríceps", Equipment: "Máquina", Description: "Construção de força e massa com segurança lombar."},
		{Name: "Leg Press Horizontal", MuscleGroup: "Quadríceps", Equipment: "Máquina", Description: "Variação de empurrada horizontal guiada."},
		{Name: "Cadeira Extensora", MuscleGroup: "Quadríceps", Equipment: "Máquina", Description: "Isolamento direto das quatro cabeças do quadríceps."},
		{Name: "Agachamento Búlgaro", MuscleGroup: "Quadríceps", Equipment: "Halteres e Banco", Description: "Excelente trabalho unilateral para força e estabilidade."},
		{Name: "Avanço / Passada com Halteres", MuscleGroup: "Quadríceps", Equipment: "Halteres", Description: "Exercício dinâmico e funcional para pernas e glúteos."},
		{Name: "Agachamento Hack", MuscleGroup: "Quadríceps", Equipment: "Máquina Hack", Description: "Sobrecarga profunda e guiada de quadríceps."},
		{Name: "Agachamento Sissy", MuscleGroup: "Quadríceps", Equipment: "Peso Corporal / Apoio", Description: "Isolamento extremo de quadríceps com alongamento."},

		// === ISQUIOTIBIAIS / POSTERIOR DE COXA ===
		{Name: "Mesa Flexora", MuscleGroup: "Posterior de Coxa", Equipment: "Máquina", Description: "Flexão de joelhos com quadril estendido."},
		{Name: "Cadeira Flexora", MuscleGroup: "Posterior de Coxa", Equipment: "Máquina", Description: "Maior alongamento dos isquiotibiais pelo quadril flexionado."},
		{Name: "Flexora Vertical Unilateral", MuscleGroup: "Posterior de Coxa", Equipment: "Máquina", Description: "Correção de assimetrias entre as pernas."},
		{Name: "Stiff com Barra", MuscleGroup: "Posterior de Coxa", Equipment: "Barra", Description: "Alongamento e tensão máxima na cadeia posterior."},
		{Name: "Stiff com Halteres", MuscleGroup: "Posterior de Coxa", Equipment: "Halteres", Description: "Permite trajeto anatômico rente às pernas."},
		{Name: "Good Morning (Bom Dia)", MuscleGroup: "Posterior de Coxa", Equipment: "Barra", Description: "Fortalecimento de lombar, glúteos e isquiotibiais."},
		{Name: "Levantamento Terra Romeno (RDL)", MuscleGroup: "Posterior de Coxa", Equipment: "Barra", Description: "Foco concentrado na fase excêntrica de posterior."},

		// === GLÚTEOS ===
		{Name: "Elevação Pélvica com Barra (Hip Thrust)", MuscleGroup: "Glúteos", Equipment: "Barra e Banco", Description: "Exercício número um para hipertrofia do glúteo máximo."},
		{Name: "Elevação Pélvica na Máquina", MuscleGroup: "Glúteos", Equipment: "Máquina", Description: "Movimento guiado para sobrecarga confortável."},
		{Name: "Cadeira Abdutora", MuscleGroup: "Glúteos", Equipment: "Máquina", Description: "Foco no glúteo médio e estabilidade pélvica."},
		{Name: "Glúteo 4 Apoios na Polia (Cabo)", MuscleGroup: "Glúteos", Equipment: "Cabo e Caneleira", Description: "Extensão de quadril com tensão contínua."},
		{Name: "Glúteo Coice na Máquina", MuscleGroup: "Glúteos", Equipment: "Máquina", Description: "Empurrada para ativação de glúteo máximo."},
		{Name: "Agachamento Sumô com Halter", MuscleGroup: "Glúteos", Equipment: "Halter / Step", Description: "Base aberta com foco em adutores e glúteos."},

		// === ADUTORES ===
		{Name: "Cadeira Adutora", MuscleGroup: "Adutores", Equipment: "Máquina", Description: "Isolamento da musculatura interna da coxa."},
		{Name: "Adução no Cabo", MuscleGroup: "Adutores", Equipment: "Cabo e Caneleira", Description: "Trabalho dinâmico unilateral de adutores."},

		// === PANTURRILHAS ===
		{Name: "Gêmeos em Pé no Degrau", MuscleGroup: "Panturrilhas", Equipment: "Peso Corporal / Barra", Description: "Enfatiza os gastrocnêmios (músculo visível)."},
		{Name: "Gêmeos no Smith", MuscleGroup: "Panturrilhas", Equipment: "Máquina Smith e Step", Description: "Controle de carga alta em pé."},
		{Name: "Gêmeos Sentado (Cavalinho)", MuscleGroup: "Panturrilhas", Equipment: "Máquina", Description: "Foco no sóleo devido aos joelhos flexionados."},
		{Name: "Gêmeos no Leg Press", MuscleGroup: "Panturrilhas", Equipment: "Máquina Leg Press", Description: "Amplitude controlada com carga progressiva."},

		// === ABDÔMEN / CORE ===
		{Name: "Abdominal Crunch Tradicional", MuscleGroup: "Abdômen", Equipment: "Colchonete", Description: "Flexão de tronco com foco em reto abdominal."},
		{Name: "Abdominal Infra no Solo", MuscleGroup: "Abdômen", Equipment: "Colchonete", Description: "Elevação de pernas para ativação da região inferior."},
		{Name: "Abdominal Suspenso na Barra (Hanging Leg Raise)", MuscleGroup: "Abdômen", Equipment: "Barra Fixa", Description: "Desafio avançado de força de core e controle."},
		{Name: "Abdominal na Polia Alta (Cabo)", MuscleGroup: "Abdômen", Equipment: "Cabo e Corda", Description: "Permite sobrecarga progressiva com peso."},
		{Name: "Prancha Abdominal Isométrica", MuscleGroup: "Abdômen", Equipment: "Colchonete", Description: "Fortalecimento profundo de transverso e estabilização."},
		{Name: "Prancha Lateral", MuscleGroup: "Abdômen", Equipment: "Colchonete", Description: "Foco em oblíquos e controle pélvico."},
		{Name: "Abdominal Roda (Ab Wheel)", MuscleGroup: "Abdômen", Equipment: "Roda Abdominal", Description: "Extensão e retração de altíssima exigência de core."},
		{Name: "Russian Twist", MuscleGroup: "Abdômen", Equipment: "Halter / Medball", Description: "Rotação de tronco para tônus de oblíquos."},
		{Name: "Hiperextensão Lombar (Banco Romano)", MuscleGroup: "Abdômen", Equipment: "Banco Romano", Description: "Eretores da espinha e saúde da coluna."},

		// === CARDIO & FUNCIONAL ===
		{Name: "Esteira - Corrida / Caminhada Inclinada", MuscleGroup: "Cardio", Equipment: "Esteira", Description: "Condicionamento aeróbico e queima calórica."},
		{Name: "Bicicleta Ergométrica (Spinning)", MuscleGroup: "Cardio", Equipment: "Bicicleta", Description: "Baixo impacto articular e resistência cardiovascular."},
		{Name: "Elíptico / Transport", MuscleGroup: "Cardio", Equipment: "Elíptico", Description: "Sem impacto nos joelhos com ativação corporal global."},
		{Name: "Simulador de Escada", MuscleGroup: "Cardio", Equipment: "Escada Ergométrica", Description: "Alta intensidade e ativação de glúteos e pernas."},
		{Name: "Remo Seco (Rowing Machine)", MuscleGroup: "Cardio", Equipment: "Remo", Description: "Cardio completo ativando pernas, costas e core."},
		{Name: "Corda Naval (Battle Rope)", MuscleGroup: "Cardio", Equipment: "Corda Naval", Description: "Treino metabólico e condicionamento de membros superiores."},
		{Name: "Kettlebell Swing", MuscleGroup: "Funcional", Equipment: "Kettlebell", Description: "Potência de quadril e cadeia posterior."},
		{Name: "Burpee", MuscleGroup: "Funcional", Equipment: "Peso Corporal", Description: "Movimento calistênico e cardiovascular dinâmico."},
		{Name: "Box Jump (Salto no Caixote)", MuscleGroup: "Funcional", Equipment: "Caixote Plyo", Description: "Potência e explosão muscular de membros inferiores."},
	}
	return list
}

func seedWorkoutTemplates(db *gorm.DB) error {
	var count int64
	db.Model(&domain.WorkoutTemplate{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("[SEED] Inserindo biblioteca de modelos de treino...")

	pick := func(name string) uuid.UUID {
		var e domain.Exercise
		db.Where("name = ?", name).First(&e)
		return e.ID
	}

	// 1. Adaptação Iniciante (Full Body)
	t1 := domain.WorkoutTemplate{
		Title:       "Adaptação Iniciante (Full Body)",
		Goal:        "Adaptação",
		Level:       "Iniciante",
		Description: "Ficha básica de adaptação anatômica e aprendizado motor para novos alunos. Frequência sugerida: 3x na semana em dias alternados.",
	}
	if err := db.Create(&t1).Error; err != nil {
		return err
	}
	div1 := domain.WorkoutTemplateDiv{
		WorkoutTemplateID: t1.ID,
		Letter:            "A",
		Name:              "Corpo Inteiro (Full Body)",
		Description:       "Exercícios compostos básicos e máquinas com apoio guiado.",
		OrderIndex:        1,
	}
	db.Create(&div1)
	items1 := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Leg Press 45°"), Sets: 3, Reps: "12-15", LoadKg: 40, RestSeconds: 60, OrderIndex: 1, Notes: "Ajustar banco para apoiar bem a coluna"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Supino Reto com Halteres"), Sets: 3, Reps: "12", LoadKg: 8, RestSeconds: 60, OrderIndex: 2, Notes: "Manter escápulas aduzidas"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Puxada Frontal Aberta"), Sets: 3, Reps: "12", LoadKg: 25, RestSeconds: 60, OrderIndex: 3, Notes: "Puxar em direção ao peitoral superior"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Cadeira Extensora"), Sets: 3, Reps: "15", LoadKg: 20, RestSeconds: 45, OrderIndex: 4, Notes: "Pausa de 1 segundo no topo"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Elevação Lateral com Halteres"), Sets: 3, Reps: "12-15", LoadKg: 4, RestSeconds: 45, OrderIndex: 5, Notes: "Não ultrapassar a linha dos ombros"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Tríceps Pulley com Barra Reta"), Sets: 3, Reps: "12", LoadKg: 15, RestSeconds: 45, OrderIndex: 6, Notes: "Cotovelos fixos ao lado do corpo"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Rosca Direta com Barra W"), Sets: 3, Reps: "12", LoadKg: 10, RestSeconds: 45, OrderIndex: 7, Notes: "Sem balanço de tronco"},
		{WorkoutTemplateDivID: div1.ID, ExerciseID: pick("Prancha Abdominal Isométrica"), Sets: 3, Reps: "30s", LoadKg: 0, RestSeconds: 45, OrderIndex: 8, Notes: "Manter abdômen firme e alinhado"},
	}
	for _, it := range items1 {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	// 2. Hipertrofia ABC Clássico
	t2 := domain.WorkoutTemplate{
		Title:       "Hipertrofia ABC Clássico (Push / Pull / Legs)",
		Goal:        "Hipertrofia",
		Level:       "Intermediário",
		Description: "A divisão de treino mais comprovada para ganho de massa muscular. Separa empurrar, puxar e membros inferiores.",
	}
	db.Create(&t2)
	div2A := domain.WorkoutTemplateDiv{WorkoutTemplateID: t2.ID, Letter: "A", Name: "Push — Peitoral, Tríceps & Ombros", OrderIndex: 1}
	div2B := domain.WorkoutTemplateDiv{WorkoutTemplateID: t2.ID, Letter: "B", Name: "Pull — Dorsal, Bíceps & Trapézio", OrderIndex: 2}
	div2C := domain.WorkoutTemplateDiv{WorkoutTemplateID: t2.ID, Letter: "C", Name: "Legs — Quadríceps, Isquiotibiais & Glúteos", OrderIndex: 3}
	db.Create(&div2A)
	db.Create(&div2B)
	db.Create(&div2C)

	items2A := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Supino Reto com Barra"), Sets: 4, Reps: "8-10", LoadKg: 30, RestSeconds: 90, OrderIndex: 1, Notes: "Carga progressiva com boa técnica"},
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Supino Inclinado com Halteres"), Sets: 4, Reps: "10-12", LoadKg: 16, RestSeconds: 60, OrderIndex: 2, Notes: "Inclinação do banco em 30°"},
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Crucifixo Reto com Halteres"), Sets: 3, Reps: "12", LoadKg: 12, RestSeconds: 60, OrderIndex: 3},
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Desenvolvimento Militar com Halteres"), Sets: 4, Reps: "10", LoadKg: 12, RestSeconds: 60, OrderIndex: 4},
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Elevação Lateral com Halteres"), Sets: 4, Reps: "12-15", LoadKg: 6, RestSeconds: 45, OrderIndex: 5},
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Tríceps Pulley com Corda"), Sets: 4, Reps: "12", LoadKg: 20, RestSeconds: 45, OrderIndex: 6, Notes: "Abrir a corda no final"},
		{WorkoutTemplateDivID: div2A.ID, ExerciseID: pick("Tríceps Testa com Barra W"), Sets: 3, Reps: "10-12", LoadKg: 12, RestSeconds: 60, OrderIndex: 7},
	}
	for _, it := range items2A {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	items2B := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div2B.ID, ExerciseID: pick("Puxada Frontal Aberta"), Sets: 4, Reps: "10-12", LoadKg: 40, RestSeconds: 60, OrderIndex: 1},
		{WorkoutTemplateDivID: div2B.ID, ExerciseID: pick("Remada Curvada com Barra (Pronada)"), Sets: 4, Reps: "8-10", LoadKg: 30, RestSeconds: 90, OrderIndex: 2},
		{WorkoutTemplateDivID: div2B.ID, ExerciseID: pick("Remada Baixa no Cabo (Triângulo)"), Sets: 3, Reps: "10-12", LoadKg: 35, RestSeconds: 60, OrderIndex: 3},
		{WorkoutTemplateDivID: div2B.ID, ExerciseID: pick("Rosca Direta com Barra W"), Sets: 4, Reps: "10", LoadKg: 15, RestSeconds: 60, OrderIndex: 4},
		{WorkoutTemplateDivID: div2B.ID, ExerciseID: pick("Rosca Martelo com Halteres"), Sets: 3, Reps: "12", LoadKg: 10, RestSeconds: 45, OrderIndex: 5},
		{WorkoutTemplateDivID: div2B.ID, ExerciseID: pick("Rosca Scott com Barra W"), Sets: 3, Reps: "10-12", LoadKg: 12, RestSeconds: 60, OrderIndex: 6},
	}
	for _, it := range items2B {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	items2C := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div2C.ID, ExerciseID: pick("Agachamento Livre com Barra"), Sets: 4, Reps: "8-10", LoadKg: 40, RestSeconds: 120, OrderIndex: 1},
		{WorkoutTemplateDivID: div2C.ID, ExerciseID: pick("Leg Press 45°"), Sets: 4, Reps: "10-12", LoadKg: 100, RestSeconds: 90, OrderIndex: 2},
		{WorkoutTemplateDivID: div2C.ID, ExerciseID: pick("Cadeira Extensora"), Sets: 3, Reps: "12-15", LoadKg: 30, RestSeconds: 60, OrderIndex: 3},
		{WorkoutTemplateDivID: div2C.ID, ExerciseID: pick("Mesa Flexora"), Sets: 4, Reps: "10-12", LoadKg: 25, RestSeconds: 60, OrderIndex: 4},
		{WorkoutTemplateDivID: div2C.ID, ExerciseID: pick("Elevação Pélvica com Barra (Hip Thrust)"), Sets: 4, Reps: "10-12", LoadKg: 40, RestSeconds: 90, OrderIndex: 5},
		{WorkoutTemplateDivID: div2C.ID, ExerciseID: pick("Gêmeos no Leg Press"), Sets: 4, Reps: "15", LoadKg: 80, RestSeconds: 45, OrderIndex: 6},
	}
	for _, it := range items2C {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	// 3. Foco Glúteo e Membros Inferiores Feminino (AB)
	t3 := domain.WorkoutTemplate{
		Title:       "Feminino: Foco Glúteos & Pernas (AB)",
		Goal:        "Hipertrofia",
		Level:       "Intermediário",
		Description: "Protocolo especializado com ênfase máxima em glúteo máximo, médio e quadríceps, com estímulo compensatório para membros superiores.",
	}
	db.Create(&t3)
	div3A := domain.WorkoutTemplateDiv{WorkoutTemplateID: t3.ID, Letter: "A", Name: "Glúteos & Quadríceps", OrderIndex: 1}
	div3B := domain.WorkoutTemplateDiv{WorkoutTemplateID: t3.ID, Letter: "B", Name: "Posteriores, Superiores & Abdômen", OrderIndex: 2}
	db.Create(&div3A)
	db.Create(&div3B)

	items3A := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div3A.ID, ExerciseID: pick("Elevação Pélvica com Barra (Hip Thrust)"), Sets: 4, Reps: "10-12", LoadKg: 35, RestSeconds: 90, OrderIndex: 1, Notes: "Pausa de 2 segundos no topo"},
		{WorkoutTemplateDivID: div3A.ID, ExerciseID: pick("Agachamento Búlgaro"), Sets: 3, Reps: "10 cada lado", LoadKg: 8, RestSeconds: 60, OrderIndex: 2},
		{WorkoutTemplateDivID: div3A.ID, ExerciseID: pick("Leg Press 45°"), Sets: 4, Reps: "12", LoadKg: 70, RestSeconds: 60, OrderIndex: 3, Notes: "Pés na largura dos ombros"},
		{WorkoutTemplateDivID: div3A.ID, ExerciseID: pick("Cadeira Extensora"), Sets: 3, Reps: "15", LoadKg: 25, RestSeconds: 45, OrderIndex: 4},
		{WorkoutTemplateDivID: div3A.ID, ExerciseID: pick("Cadeira Abdutora"), Sets: 4, Reps: "15-20", LoadKg: 35, RestSeconds: 45, OrderIndex: 5, Notes: "Inclinar o tronco à frente"},
		{WorkoutTemplateDivID: div3A.ID, ExerciseID: pick("Gêmeos no Leg Press"), Sets: 4, Reps: "15", LoadKg: 60, RestSeconds: 45, OrderIndex: 6},
	}
	for _, it := range items3A {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	items3B := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div3B.ID, ExerciseID: pick("Mesa Flexora"), Sets: 4, Reps: "10-12", LoadKg: 25, RestSeconds: 60, OrderIndex: 1},
		{WorkoutTemplateDivID: div3B.ID, ExerciseID: pick("Puxada Frontal com Triângulo"), Sets: 3, Reps: "12", LoadKg: 25, RestSeconds: 60, OrderIndex: 2},
		{WorkoutTemplateDivID: div3B.ID, ExerciseID: pick("Supino Inclinado com Halteres"), Sets: 3, Reps: "12", LoadKg: 8, RestSeconds: 60, OrderIndex: 3},
		{WorkoutTemplateDivID: div3B.ID, ExerciseID: pick("Desenvolvimento Militar com Halteres"), Sets: 3, Reps: "12", LoadKg: 6, RestSeconds: 60, OrderIndex: 4},
		{WorkoutTemplateDivID: div3B.ID, ExerciseID: pick("Prancha Abdominal Isométrica"), Sets: 3, Reps: "45s", LoadKg: 0, RestSeconds: 45, OrderIndex: 5},
	}
	for _, it := range items3B {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	// 4. Funcional & Emagrecimento (Metabólico)
	t4 := domain.WorkoutTemplate{
		Title:       "Funcional & Queima Calórica (Metabólico)",
		Goal:        "Emagrecimento",
		Level:       "Iniciante",
		Description: "Protocolo de alta intensidade metabólica com exercícios dinâmicos e curtos tempos de descanso. Ideal para condicionamento e queima calórica.",
	}
	db.Create(&t4)
	div4A := domain.WorkoutTemplateDiv{WorkoutTemplateID: t4.ID, Letter: "A", Name: "Circuito Full Body Metabólico", OrderIndex: 1}
	db.Create(&div4A)
	items4A := []domain.WorkoutTemplateItem{
		{WorkoutTemplateDivID: div4A.ID, ExerciseID: pick("Agachamento Livre com Barra"), Sets: 4, Reps: "15", LoadKg: 20, RestSeconds: 45, OrderIndex: 1},
		{WorkoutTemplateDivID: div4A.ID, ExerciseID: pick("Flexão de Braços (Push-up)"), Sets: 3, Reps: "12", LoadKg: 0, RestSeconds: 45, OrderIndex: 2},
		{WorkoutTemplateDivID: div4A.ID, ExerciseID: pick("Remada Unilateral com Halter (Serrote)"), Sets: 3, Reps: "12", LoadKg: 12, RestSeconds: 45, OrderIndex: 3},
		{WorkoutTemplateDivID: div4A.ID, ExerciseID: pick("Desenvolvimento Militar com Halteres"), Sets: 3, Reps: "15", LoadKg: 6, RestSeconds: 45, OrderIndex: 4},
		{WorkoutTemplateDivID: div4A.ID, ExerciseID: pick("Prancha Abdominal Isométrica"), Sets: 3, Reps: "45s", LoadKg: 0, RestSeconds: 30, OrderIndex: 5},
	}
	for _, it := range items4A {
		if it.ExerciseID != uuid.Nil {
			db.Create(&it)
		}
	}

	log.Println("[SEED] Biblioteca de modelos de treino populada com sucesso!")
	return nil
}
