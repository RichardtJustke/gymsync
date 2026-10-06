# 🏋️ GymSync — Plataforma Integrada de Gestão de Academias

O **GymSync** é um ecossistema SaaS para academias que conecta de ponta a ponta:
- **🏢 Academia / Gestores:** Visão gerencial completa, relatórios de faturamento, controle de inadimplência, frequência e importador inteligente de clientes via planilhas (Excel/CSV).
- **💪 Professores / Personais:** Prescrição rápida de treinos a partir de modelos prontos, histórico de evolução do aluno e acompanhamento em tempo real dos alunos presentes.
- **📱 Alunos:** Acompanhamento de fichas com cronômetro de descanso embutido, histórico de cargas, check-in por QR Code e pagamentos simulados (Boletos e PIX).

---

## 🛠️ Stack Tecnológica

- **Backend:** Golang (Go 1.22+), GORM, Chi Router, Excelize, JWT Authentication.
- **Banco de Dados:** PostgreSQL 16 (Multi-tenant por isolamento de `GymID`).
- **Frontend & PWA:** Next.js (React 19 / App Router), Tailwind CSS, Lucide Icons, Shadcn UI pattern.
- **Notificações:** Arquitetura pronta para integração com a **Evolution API** (WhatsApp).

---

## 🚀 Como Iniciar Localmente

### 1. Pré-requisitos
- Docker & Docker Compose
- Go 1.22+
- Node.js 18+ & npm

### 2. Subir o Banco de Dados
```bash
docker compose up -d
```

### 3. Backend (Go)
```bash
cd backend
go run cmd/api/main.go
```
A API estará disponível em `http://localhost:8080`.

### 4. Frontend (Next.js)
```bash
cd frontend
npm install
npm run dev
```
Acesse `http://localhost:3000`.
