package handler

import (
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// StaffHandler equipe: cadastro, cargos, permissões, horas/aula e folha
type StaffHandler struct {
	db    *gorm.DB
	perms *service.Permissions
	team  *service.TeamService
}

func NewStaffHandler(db *gorm.DB, perms *service.Permissions, team *service.TeamService) *StaffHandler {
	return &StaffHandler{db: db, perms: perms, team: team}
}

var roleLabels = map[domain.Role]string{
	domain.RoleAdmin: "Dono(a)", domain.RoleGerente: "Gerente", domain.RoleRecepcao: "Recepção",
	domain.RoleConsultor: "Consultor(a)", domain.RoleProfessor: "Professor(a)",
}

// StaffMember pessoa da equipe com perfil e números rápidos
type StaffMember struct {
	domain.User
	Profile      *domain.StaffProfile `json:"profile"`
	RoleLabel    string               `json:"role_label"`
	ClassesMonth int64                `json:"classes_month"`
	SalesMonth   int64                `json:"sales_month"`
	LastAction   *time.Time           `json:"last_action"`
}

// ListStaff ?inactive=1 inclui desativados
func (h *StaffHandler) ListStaff(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var users []domain.User
	q := h.db.Where("gym_id = ? AND role IN ?", gymID, domain.StaffRoles)
	if r.URL.Query().Get("inactive") != "1" {
		q = q.Where("active = ?", true)
	}
	q.Order("active desc, name asc").Find(&users)

	ids := make([]uuid.UUID, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	profiles := map[uuid.UUID]domain.StaffProfile{}
	var plist []domain.StaffProfile
	if len(ids) > 0 {
		h.db.Where("user_id IN ?", ids).Find(&plist)
	}
	for _, p := range plist {
		profiles[p.UserID] = p
	}
	month, _ := service.ParseMonth("")
	end := month.AddDate(0, 1, 0)

	out := make([]StaffMember, 0, len(users))
	for _, u := range users {
		m := StaffMember{User: u, RoleLabel: roleLabels[u.Role]}
		if p, ok := profiles[u.ID]; ok {
			pp := p
			m.Profile = &pp
		}
		h.db.Model(&domain.ClassSession{}).Where("teacher_id = ? AND status = ? AND starts_at >= ? AND starts_at < ?", u.ID, domain.SessionScheduled, month.UTC(), end.UTC()).Count(&m.ClassesMonth)
		h.db.Model(&domain.Sale{}).Where("seller_id = ? AND status = ? AND created_at >= ? AND created_at < ?", u.ID, domain.SaleActive, month, end).Count(&m.SalesMonth)
		var last domain.AuditLog
		if h.db.Select("created_at").Where("user_id = ?", u.ID).Order("created_at desc").First(&last).Error == nil {
			t := last.CreatedAt
			m.LastAction = &t
		}
		out = append(out, m)
	}
	respondJSON(w, http.StatusOK, out)
}

type StaffProfileRequest struct {
	HireDate          string         `json:"hire_date"` // AAAA-MM-DD
	PayType           domain.PayType `json:"pay_type"`
	BaseSalary        float64        `json:"base_salary"`
	HourlyRate        float64        `json:"hourly_rate"`
	CommissionPercent float64        `json:"commission_percent"`
	CommissionFixed   float64        `json:"commission_fixed"`
	PixKey            string         `json:"pix_key"`
	Notes             string         `json:"notes"`
}

type StaffRequest struct {
	Name     string               `json:"name"`
	Email    string               `json:"email"`
	Document string               `json:"document"`
	Phone    string               `json:"phone"`
	Role     domain.Role          `json:"role"`
	JobTitle string               `json:"job_title"`
	Profile  *StaffProfileRequest `json:"profile"`
}

func (req *StaffProfileRequest) apply(p *domain.StaffProfile) error {
	if req.BaseSalary < 0 || req.HourlyRate < 0 || req.CommissionFixed < 0 || req.CommissionPercent < 0 || req.CommissionPercent > 100 {
		return &service.BusinessError{Msg: "Valores de remuneração inválidos"}
	}
	switch req.PayType {
	case domain.PayFixed, domain.PayHourly, domain.PayMixed:
		p.PayType = req.PayType
	case "":
		if p.PayType == "" {
			p.PayType = domain.PayFixed
		}
	default:
		return &service.BusinessError{Msg: "Tipo de remuneração inválido"}
	}
	p.HireDate = nil
	if req.HireDate != "" {
		t, err := time.ParseInLocation("2006-01-02", req.HireDate, config.Loc())
		if err != nil {
			return &service.BusinessError{Msg: "Data de admissão inválida"}
		}
		p.HireDate = &t
	}
	p.BaseSalary, p.HourlyRate = req.BaseSalary, req.HourlyRate
	p.CommissionPercent, p.CommissionFixed = req.CommissionPercent, req.CommissionFixed
	p.PixKey, p.Notes = strings.TrimSpace(req.PixKey), req.Notes
	return nil
}

func validStaffRole(role domain.Role) bool {
	return domain.IsStaffRole(role)
}

// CreateStaff cadastra alguém da equipe com senha provisória
func (h *StaffHandler) CreateStaff(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req StaffRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validStaffRole(req.Role) {
		respondError(w, http.StatusBadRequest, "Cargo inválido")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		respondError(w, http.StatusBadRequest, "E-mail é obrigatório para a equipe")
		return
	}
	var user *domain.User
	var temp string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var err error
		user, temp, err = service.CreateUser(tx, gymID, service.NewAccountInput{
			Name: req.Name, Email: req.Email, Document: req.Document, Phone: req.Phone, Role: req.Role,
		})
		if err != nil {
			return err
		}
		user.JobTitle = strings.TrimSpace(req.JobTitle)
		if err := tx.Model(user).Update("job_title", user.JobTitle).Error; err != nil {
			return err
		}
		p := domain.StaffProfile{GymID: gymID, UserID: user.ID}
		if req.Profile != nil {
			if err := req.Profile.apply(&p); err != nil {
				return err
			}
		} else {
			p.PayType = domain.PayFixed
		}
		return tx.Create(&p).Error
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "staff.create", "staff", user.ID, "Cadastrou %s como %s", user.Name, roleLabels[user.Role])
	respondJSON(w, http.StatusCreated, map[string]interface{}{"user": user, "temp_password": temp, "login": user.Email})
}

func (h *StaffHandler) findStaff(w http.ResponseWriter, r *http.Request) (*domain.User, bool) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	var u domain.User
	if err := h.db.First(&u, "id = ? AND gym_id = ? AND role IN ?", id, gymID, domain.StaffRoles).Error; err != nil {
		respondError(w, http.StatusNotFound, "Pessoa não encontrada na equipe")
		return nil, false
	}
	return &u, true
}

// countOwners quantos donos ativos restam (a academia nunca pode ficar sem)
func (h *StaffHandler) countOwners(gymID uuid.UUID, except uuid.UUID) int64 {
	var n int64
	h.db.Model(&domain.User{}).Where("gym_id = ? AND role = ? AND active = ? AND id <> ?", gymID, domain.RoleAdmin, true, except).Count(&n)
	return n
}

// UpdateStaff dados, cargo e remuneração
func (h *StaffHandler) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	u, ok := h.findStaff(w, r)
	if !ok {
		return
	}
	var req StaffRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role != "" && !validStaffRole(req.Role) {
		respondError(w, http.StatusBadRequest, "Cargo inválido")
		return
	}
	// Só o dono mexe em dono e promove alguém a dono
	if (u.Role == domain.RoleAdmin || req.Role == domain.RoleAdmin) && claims.Role != domain.RoleAdmin {
		respondError(w, http.StatusForbidden, "Somente o(a) dono(a) pode alterar acessos de dono")
		return
	}
	if u.Role == domain.RoleAdmin && req.Role != "" && req.Role != domain.RoleAdmin && h.countOwners(u.GymID, u.ID) == 0 {
		respondError(w, http.StatusBadRequest, "A academia precisa de pelo menos um(a) dono(a)")
		return
	}
	oldRole := u.Role
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if strings.TrimSpace(req.Name) != "" {
			u.Name = strings.TrimSpace(req.Name)
		}
		u.Email = strings.ToLower(strings.TrimSpace(req.Email))
		u.Document = cleanDocument(req.Document)
		u.Phone = req.Phone
		u.JobTitle = strings.TrimSpace(req.JobTitle)
		if u.Email == "" {
			return &service.BusinessError{Msg: "E-mail é obrigatório para a equipe"}
		}
		if err := service.EnsureLoginAvailable(tx, u.Email, u.Document, &u.ID); err != nil {
			return err
		}
		if req.Role != "" {
			u.Role = req.Role
		}
		if err := tx.Model(u).Select("name", "email", "document", "phone", "job_title", "role").Updates(u).Error; err != nil {
			return err
		}
		if req.Profile != nil {
			var p domain.StaffProfile
			exists := tx.Where("user_id = ?", u.ID).First(&p).Error == nil
			p.GymID, p.UserID = u.GymID, u.ID
			if err := req.Profile.apply(&p); err != nil {
				return err
			}
			if exists {
				return tx.Save(&p).Error
			}
			return tx.Create(&p).Error
		}
		return nil
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	if oldRole != u.Role {
		// Saiu de professor(a): deixa as aulas futuras sem professor(a) para a gestão redistribuir
		if oldRole == domain.RoleProfessor {
			h.unassignFuture(u.GymID, u.ID)
		}
		audit(h.db, r, "staff.role", "staff", u.ID, "Mudou o cargo de %s: %s → %s", u.Name, roleLabels[oldRole], roleLabels[u.Role])
	} else {
		audit(h.db, r, "staff.update", "staff", u.ID, "Atualizou o cadastro de %s", u.Name)
	}
	respondJSON(w, http.StatusOK, u)
}

func (h *StaffHandler) unassignFuture(gymID, userID uuid.UUID) int64 {
	h.db.Model(&domain.ClassSchedule{}).Where("gym_id = ? AND teacher_id = ?", gymID, userID).Update("teacher_id", nil)
	res := h.db.Model(&domain.ClassSession{}).Where("gym_id = ? AND teacher_id = ? AND starts_at > ?", gymID, userID, time.Now()).Update("teacher_id", nil)
	return res.RowsAffected
}

// SetActive desativa (encerra o acesso na hora) ou reativa alguém da equipe
func (h *StaffHandler) SetActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.GetUserFromContext(r.Context())
		u, ok := h.findStaff(w, r)
		if !ok {
			return
		}
		if u.ID == claims.UserID {
			respondError(w, http.StatusBadRequest, "Você não pode desativar o próprio acesso")
			return
		}
		if u.Role == domain.RoleAdmin && claims.Role != domain.RoleAdmin {
			respondError(w, http.StatusForbidden, "Somente o(a) dono(a) pode alterar acessos de dono")
			return
		}
		if !active && u.Role == domain.RoleAdmin && h.countOwners(u.GymID, u.ID) == 0 {
			respondError(w, http.StatusBadRequest, "A academia precisa de pelo menos um(a) dono(a)")
			return
		}
		updates := map[string]interface{}{"active": active}
		if !active {
			updates["token_version"] = gorm.Expr("token_version + 1")
		}
		if err := h.db.Model(&domain.User{}).Where("id = ?", u.ID).Updates(updates).Error; err != nil {
			respondErr(w, err)
			return
		}
		resp := map[string]interface{}{"success": true}
		if active {
			audit(h.db, r, "staff.activate", "staff", u.ID, "Reativou o acesso de %s", u.Name)
		} else {
			uncovered := int64(0)
			if u.Role == domain.RoleProfessor {
				uncovered = h.unassignFuture(u.GymID, u.ID)
			}
			h.db.Where("user_id = ?", u.ID).Delete(&domain.DeviceToken{})
			resp["uncovered_classes"] = uncovered
			audit(h.db, r, "staff.deactivate", "staff", u.ID, "Desativou o acesso de %s (%d aula(s) futuras ficaram sem professor)", u.Name, uncovered)
		}
		respondJSON(w, http.StatusOK, resp)
	}
}

// ResetStaffPassword nova senha provisória e encerra as sessões abertas
func (h *StaffHandler) ResetStaffPassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	u, ok := h.findStaff(w, r)
	if !ok {
		return
	}
	if u.Role == domain.RoleAdmin && claims.Role != domain.RoleAdmin {
		respondError(w, http.StatusForbidden, "Somente o(a) dono(a) pode alterar acessos de dono")
		return
	}
	temp := service.TempPassword()
	hash, _ := bcrypt.GenerateFromPassword([]byte(temp), bcrypt.DefaultCost)
	if err := h.db.Model(&domain.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"password_hash": string(hash), "first_access": true, "token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "staff.password", "staff", u.ID, "Gerou nova senha provisória para %s", u.Name)
	respondJSON(w, http.StatusOK, map[string]string{"login": u.Email, "temp_password": temp})
}

/* --------------------------- Permissões --------------------------- */

type RolePermissionsView struct {
	Role        domain.Role         `json:"role"`
	Label       string              `json:"label"`
	Editable    bool                `json:"editable"`
	Custom      bool                `json:"custom"` // diferente do padrão
	Permissions []domain.Permission `json:"permissions"`
	Members     int64               `json:"members"`
}

func (h *StaffHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var custom []domain.RolePermission
	h.db.Where("gym_id = ?", gymID).Find(&custom)
	isCustom := map[domain.Role]bool{}
	for _, c := range custom {
		isCustom[c.Role] = true
	}
	roles := []RolePermissionsView{}
	for _, role := range domain.StaffRoles {
		v := RolePermissionsView{Role: role, Label: roleLabels[role], Editable: role != domain.RoleAdmin, Custom: isCustom[role], Permissions: h.perms.List(gymID, role)}
		h.db.Model(&domain.User{}).Where("gym_id = ? AND role = ? AND active = ?", gymID, role, true).Count(&v.Members)
		roles = append(roles, v)
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"catalog": domain.PermissionCatalog, "roles": roles})
}

type SetPermissionsRequest struct {
	Permissions []domain.Permission `json:"permissions"`
	Reset       bool                `json:"reset"`
}

func (h *StaffHandler) SetPermissions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	role := domain.Role(strings.ToUpper(chiParam(r, "role")))
	var req SetPermissionsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// Ninguém tira de si mesmo o acesso a esta tela
	if role == claims.Role && !req.Reset && !containsPerm(req.Permissions, domain.PermStaff) {
		respondError(w, http.StatusBadRequest, "Você não pode remover do seu próprio cargo o acesso à equipe")
		return
	}
	var err error
	if req.Reset {
		if !domain.IsStaffRole(role) || role == domain.RoleAdmin {
			err = &service.BusinessError{Msg: "Este cargo não tem permissões configuráveis"}
		} else {
			err = h.perms.Reset(claims.GymID, role)
		}
	} else {
		err = h.perms.Set(claims.GymID, role, req.Permissions)
	}
	if err != nil {
		respondErr(w, err)
		return
	}
	list := h.perms.List(claims.GymID, role)
	labels := make([]string, 0, len(list))
	for _, p := range list {
		for _, info := range domain.PermissionCatalog {
			if info.Key == p {
				labels = append(labels, info.Label)
			}
		}
	}
	summary := "Restaurou as permissões padrão de " + roleLabels[role]
	if !req.Reset {
		summary = "Alterou as permissões de " + roleLabels[role] + ": " + strings.Join(labels, ", ")
	}
	audit(h.db, r, "permissions.update", "permissions", uuid.Nil, "%s", summary)
	respondJSON(w, http.StatusOK, map[string]interface{}{"role": role, "permissions": list})
}

func containsPerm(list []domain.Permission, p domain.Permission) bool {
	for _, x := range list {
		if x == p {
			return true
		}
	}
	return false
}

/* --------------------------- Horas e folha --------------------------- */

func (h *StaffHandler) monthParam(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	month, err := service.ParseMonth(r.URL.Query().Get("month"))
	if err != nil {
		respondErr(w, err)
		return time.Time{}, false
	}
	return month, true
}

func (h *StaffHandler) Hours(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	month, ok := h.monthParam(w, r)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"month": month.Format("2006-01"), "rows": h.team.Hours(gymID, month)})
}

func (h *StaffHandler) Payroll(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	month, ok := h.monthParam(w, r)
	if !ok {
		return
	}
	rows := h.team.Payroll(gymID, month)
	var total, paid float64
	for _, row := range rows {
		total += row.Total
		if row.Status == domain.PayrollPaid {
			paid += row.Total
		}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"month": month.Format("2006-01"), "rows": rows, "total": total, "paid": paid})
}

type PayrollRequest struct {
	Month      string  `json:"month"`
	Bonus      float64 `json:"bonus"`
	Deductions float64 `json:"deductions"`
	Notes      string  `json:"notes"`
	MarkPaid   bool    `json:"mark_paid"`
	Reopen     bool    `json:"reopen"`
}

// SavePayroll ajustes (bônus/descontos) e pagamento do fechamento de uma pessoa
func (h *StaffHandler) SavePayroll(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	u, ok := h.findStaffAny(w, r)
	if !ok {
		return
	}
	var req PayrollRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	month, err := service.ParseMonth(req.Month)
	if err != nil {
		respondErr(w, err)
		return
	}
	row, err := h.team.SavePayroll(gymID, u.ID, month, service.PayrollAdjust{
		Bonus: req.Bonus, Deductions: req.Deductions, Notes: req.Notes, MarkPaid: req.MarkPaid, Reopen: req.Reopen,
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	switch {
	case req.Reopen:
		audit(h.db, r, "payroll.reopen", "payroll", u.ID, "Reabriu o fechamento de %s (%s)", u.Name, month.Format("01/2006"))
	case req.MarkPaid:
		audit(h.db, r, "payroll.pay", "payroll", u.ID, "Marcou como pago o fechamento de %s (%s): R$ %s", u.Name, month.Format("01/2006"), service.FormatBRL(row.Total))
	default:
		audit(h.db, r, "payroll.adjust", "payroll", u.ID, "Ajustou o fechamento de %s (%s): bônus R$ %s, descontos R$ %s", u.Name, month.Format("01/2006"), service.FormatBRL(req.Bonus), service.FormatBRL(req.Deductions))
	}
	respondJSON(w, http.StatusOK, row)
}

// findStaffAny inclui quem já saiu (o fechamento do último mês ainda precisa ser pago)
func (h *StaffHandler) findStaffAny(w http.ResponseWriter, r *http.Request) (*domain.User, bool) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return nil, false
	}
	var u domain.User
	if err := h.db.Unscoped().First(&u, "id = ? AND gym_id = ? AND role IN ?", id, gymID, domain.StaffRoles).Error; err != nil {
		respondError(w, http.StatusNotFound, "Pessoa não encontrada na equipe")
		return nil, false
	}
	return &u, true
}
