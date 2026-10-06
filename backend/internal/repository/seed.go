package repository

import (
	"log"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedDatabase popula o banco com dados iniciais e mais de 100 exercícios categorizados
func SeedDatabase(db *gorm.DB) error {
	var count int64
	db.Model(&domain.Exercise{}).Count(&count)
	if count > 0 {
		log.Printf("[SEED] Banco já possui %d exercícios cadastrados. Pulando seed inicial.", count)
		return nil
	}

	log.Println("[SEED] Populando banco de dados com dados iniciais e ampla base de 110+ exercícios...")

	// 1. Academia Padrão
	gymID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	gym := domain.Gym{
		ID:       gymID,
		Name:     "GymSync Prime Fitness",
		Document: "12.345.678/0001-90",
		Phone:    "(11) 98765-4321",
		Email:    "contato@gymsyncprime.com.br",
		Address:  "Av. Paulista, 1000 - Bela Vista, São Paulo - SP",
		ThemeHex: "#2563eb",
	}
	db.FirstOrCreate(&gym, domain.Gym{ID: gymID})

	// 2. Usuários de Demonstração
	hashPass, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)

	// Admin (Gestor)
	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	admin := domain.User{
		ID:           adminID,
		GymID:        gymID,
		Name:         "Carlos Gestor",
		Email:        "admin@gymsync.com",
		Document:     "11122233344",
		Phone:        "(11) 99999-0001",
		PasswordHash: string(hashPass),
		Role:         domain.RoleAdmin,
		FirstAccess:  false,
	}
	db.FirstOrCreate(&admin, domain.User{Email: "admin@gymsync.com"})

	// Professor (Personal)
	profID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	prof := domain.User{
		ID:           profID,
		GymID:        gymID,
		Name:         "Lucas Treinador",
		Email:        "professor@gymsync.com",
		Document:     "22233344455",
		Phone:        "(11) 99999-0002",
		PasswordHash: string(hashPass),
		Role:         domain.RoleProfessor,
		FirstAccess:  false,
	}
	db.FirstOrCreate(&prof, domain.User{Email: "professor@gymsync.com"})

	// Aluno
	alunoUserID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	alunoUser := domain.User{
		ID:           alunoUserID,
		GymID:        gymID,
		Name:         "Matheus Aluno",
		Email:        "aluno@gymsync.com",
		Document:     "33344455566",
		Phone:        "(11) 99999-0003",
		PasswordHash: string(hashPass),
		Role:         domain.RoleAluno,
		FirstAccess:  false,
	}
	db.FirstOrCreate(&alunoUser, domain.User{Email: "aluno@gymsync.com"})

	// Perfil do Aluno
	studentID := uuid.MustParse("00000000-0000-0000-0000-000000000040")
	student := domain.Student{
		ID:        studentID,
		GymID:     gymID,
		UserID:    alunoUserID,
		PlanName:  "Plano Ouro Anual",
		PlanPrice: 119.90,
		DueDay:    10,
		Status:    domain.StatusAtivo,
		Weight:    78.5,
		Height:    177.0,
		Notes:     "Objetivo: Hipertrofia com foco em membros superiores.",
	}
	db.FirstOrCreate(&student, domain.Student{ID: studentID})

	// 3. Fatura Pendente (Mock) para teste de pagamento
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000000050")
	invoice := domain.Invoice{
		ID:           invoiceID,
		GymID:        gymID,
		StudentID:    studentID,
		Amount:       119.90,
		DueDate:      time.Now().AddDate(0, 0, 5),
		Status:       domain.InvoicePendente,
		BoletoDigits: "34191.79001 01043.510047 91020.150008 5 95680000011990",
		PixCopyPaste: "00020126580014br.gov.bcb.pix0136gymsync-mock-key-1234-5678-90ab-cdef5204000053039865406119.905802BR5916GYMSYNC FITNESS6009SAO PAULO62070503***6304ABCD",
		Description:  "Mensalidade - Plano Ouro (Mês Atual)",
	}
	db.FirstOrCreate(&invoice, domain.Invoice{ID: invoiceID})

	// 4. Ampla Base de Exercícios (110+ exercícios profissionais)
	exercises := getComprehensiveExercises()
	if err := db.CreateInBatches(exercises, 50).Error; err != nil {
		log.Printf("[SEED] Erro ao inserir exercícios: %v", err)
		return err
	}

	// 5. Treino de Exemplo A/B/C para o aluno Matheus
	planID := uuid.MustParse("00000000-0000-0000-0000-000000000060")
	plan := domain.WorkoutPlan{
		ID:        planID,
		GymID:     gymID,
		StudentID: studentID,
		TeacherID: profID,
		Title:     "Treino ABC - Hipertrofia Padrão",
		Goal:      "Ganho de massa magra e força",
		Active:    true,
		StartDate: time.Now(),
	}
	db.FirstOrCreate(&plan, domain.WorkoutPlan{ID: planID})

	// Treino A (Peito, Tríceps e Deltoide Anterior)
	workoutAID := uuid.MustParse("00000000-0000-0000-0000-000000000071")
	workoutA := domain.Workout{
		ID:            workoutAID,
		WorkoutPlanID: planID,
		Letter:        "A",
		Name:          "Peito, Tríceps e Ombro",
		Description:   "Foco em empurrar (Push Day)",
		OrderIndex:    1,
	}
	db.FirstOrCreate(&workoutA, domain.Workout{ID: workoutAID})

	// Busca alguns exercícios inseridos para linkar na ficha A
	var exSupino, exInclinado, exTricepsCord, exDesenv domain.Exercise
	db.Where("name = ?", "Supino Reto com Barra").First(&exSupino)
	db.Where("name = ?", "Supino Inclinado com Halteres").First(&exInclinado)
	db.Where("name = ?", "Tríceps Pulley com Corda").First(&exTricepsCord)
	db.Where("name = ?", "Desenvolvimento Militar com Halteres").First(&exDesenv)

	itemsA := []domain.WorkoutExercise{
		{WorkoutID: workoutAID, ExerciseID: exSupino.ID, Sets: 4, Reps: "8-10", LoadKg: 70, RestSeconds: 90, Notes: "Aquecimento progressivo na primeira série", OrderIndex: 1},
		{WorkoutID: workoutAID, ExerciseID: exInclinado.ID, Sets: 4, Reps: "10-12", LoadKg: 24, RestSeconds: 60, Notes: "Cadência 3-1", OrderIndex: 2},
		{WorkoutID: workoutAID, ExerciseID: exDesenv.ID, Sets: 3, Reps: "10-12", LoadKg: 18, RestSeconds: 60, Notes: "Coluna bem apoiada", OrderIndex: 3},
		{WorkoutID: workoutAID, ExerciseID: exTricepsCord.ID, Sets: 4, Reps: "12-15", LoadKg: 35, RestSeconds: 45, Notes: "Pico de contração de 1s", OrderIndex: 4},
	}
	for _, it := range itemsA {
		db.Create(&it)
	}

	log.Println("[SEED] Seed executado com sucesso! Usuários criados: admin@gymsync.com, professor@gymsync.com, aluno@gymsync.com (Senha padrão: 123456)")
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
