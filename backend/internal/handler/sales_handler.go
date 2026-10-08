package handler

import (
	"net/http"
	"strings"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SalesHandler vendas, metas e ranking da equipe comercial
type SalesHandler struct {
	db    *gorm.DB
	team  *service.TeamService
	perms *service.Permissions
}

func NewSalesHandler(db *gorm.DB, team *service.TeamService, perms *service.Permissions) *SalesHandler {
	return &SalesHandler{db: db, team: team, perms: perms}
}

func (h *SalesHandler) seesAll(r *http.Request) bool {
	c := middleware.GetUserFromContext(r.Context())
	return h.perms.Has(c.GymID, c.Role, domain.PermGoals)
}

// ListSales ?month=&seller_id= (sem "metas e comissões" cada um vê só as próprias)
func (h *SalesHandler) ListSales(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	month, err := service.ParseMonth(r.URL.Query().Get("month"))
	if err != nil {
		respondErr(w, err)
		return
	}
	q := h.db.Preload("Seller", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Select("id", "name", "role") }).
		Preload("Student.User").
		Where("gym_id = ? AND created_at >= ? AND created_at < ?", claims.GymID, month, month.AddDate(0, 1, 0))
	if !h.seesAll(r) {
		q = q.Where("seller_id = ?", claims.UserID)
	} else if sid, err := uuid.Parse(r.URL.Query().Get("seller_id")); err == nil {
		q = q.Where("seller_id = ?", sid)
	}
	var list []domain.Sale
	q.Order("created_at desc").Limit(500).Find(&list)
	respondJSON(w, http.StatusOK, list)
}

type SaleRequest struct {
	SellerID    *uuid.UUID      `json:"seller_id"`
	StudentID   *uuid.UUID      `json:"student_id"`
	PlanID      *uuid.UUID      `json:"plan_id"`
	Kind        domain.SaleKind `json:"kind"`
	Amount      float64         `json:"amount"`
	Description string          `json:"description"`
	ApplyPlan   bool            `json:"apply_plan"` // renovação/upgrade: também troca o plano do aluno
}

// CreateSale registra venda (renovação, upgrade, avulsa). Sem "metas e comissões", só em nome próprio.
func (h *SalesHandler) CreateSale(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req SaleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	seller := claims.UserID
	if req.SellerID != nil && *req.SellerID != uuid.Nil {
		if *req.SellerID != claims.UserID && !h.seesAll(r) {
			respondError(w, http.StatusForbidden, "Você só pode registrar vendas em seu nome")
			return
		}
		seller = *req.SellerID
	}
	var sale *domain.Sale
	var studentName string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var err error
		sale, err = h.team.RegisterSale(tx, claims.GymID, service.SaleInput{
			SellerID: seller, StudentID: req.StudentID, PlanID: req.PlanID, Kind: req.Kind,
			Amount: req.Amount, Description: req.Description, CreatedBy: claims.UserID,
		})
		if err != nil {
			return err
		}
		if req.StudentID != nil {
			var st domain.Student
			if err := tx.Preload("User").First(&st, "id = ?", *req.StudentID).Error; err != nil {
				return err
			}
			studentName = st.User.Name
			if req.ApplyPlan && req.PlanID != nil {
				if err := service.ApplyPlan(tx, claims.GymID, &st, req.PlanID); err != nil {
					return err
				}
				if st.Status == domain.StatusCancelado {
					st.Status, st.CanceledAt, st.CancelReason = domain.StatusAtivo, nil, ""
				}
				return tx.Omit("User", "Plan").Save(&st).Error
			}
		}
		return nil
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	who := ""
	if studentName != "" {
		who = " — " + studentName
	}
	audit(h.db, r, "sale.create", "sale", sale.ID, "Registrou venda (%s) de R$ %s%s", strings.ToLower(string(sale.Kind)), service.FormatBRL(sale.Amount), who)
	respondJSON(w, http.StatusCreated, sale)
}

type ReverseSaleRequest struct {
	Reason string `json:"reason"`
}

// ReverseSale estorna a venda (sai do ranking e da comissão)
func (h *SalesHandler) ReverseSale(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req ReverseSaleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var sale domain.Sale
	if err := h.db.First(&sale, "id = ? AND gym_id = ?", id, claims.GymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Venda não encontrada")
		return
	}
	if sale.Status == domain.SaleReversed {
		respondError(w, http.StatusBadRequest, "Venda já estornada")
		return
	}
	// Fechamento pago congela a comissão: estornar depois não muda o que foi pago
	var paid int64
	h.db.Model(&domain.PayrollEntry{}).Where("user_id = ? AND month = ? AND status = ?", sale.SellerID, service.MonthRef(sale.CreatedAt), domain.PayrollPaid).Count(&paid)
	if paid > 0 {
		respondError(w, http.StatusBadRequest, "A comissão desta venda já foi paga no fechamento do mês. Reabra o fechamento antes de estornar.")
		return
	}
	h.db.Model(&sale).Update("status", domain.SaleReversed)
	audit(h.db, r, "sale.reverse", "sale", sale.ID, "Estornou venda de R$ %s. Motivo: %s", service.FormatBRL(sale.Amount), strings.TrimSpace(req.Reason))
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Ranking do mês. Comissões de outras pessoas só aparecem para quem tem "metas e comissões".
func (h *SalesHandler) Ranking(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	month, err := service.ParseMonth(r.URL.Query().Get("month"))
	if err != nil {
		respondErr(w, err)
		return
	}
	res := h.team.Ranking(claims.GymID, month)
	if !h.seesAll(r) {
		for i := range res.Rows {
			if res.Rows[i].UserID != claims.UserID {
				res.Rows[i].Commission = 0
			}
		}
		res.TotalComm = 0
	}
	respondJSON(w, http.StatusOK, res)
}

type GoalRequest struct {
	UserID        *uuid.UUID `json:"user_id"` // vazio = meta da academia
	Month         string     `json:"month"`
	TargetSales   int        `json:"target_sales"`
	TargetRevenue float64    `json:"target_revenue"`
}

// SetGoal cria/atualiza a meta do mês (zerar os dois valores remove a meta)
func (h *SalesHandler) SetGoal(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req GoalRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	month, err := service.ParseMonth(req.Month)
	if err != nil {
		respondErr(w, err)
		return
	}
	if req.TargetSales < 0 || req.TargetRevenue < 0 {
		respondError(w, http.StatusBadRequest, "Meta inválida")
		return
	}
	ref := month.Format("2006-01")
	who := "da academia"
	q := h.db.Where("gym_id = ? AND month = ?", claims.GymID, ref)
	if req.UserID != nil && *req.UserID != uuid.Nil {
		var u domain.User
		if err := h.db.First(&u, "id = ? AND gym_id = ? AND role IN ?", *req.UserID, claims.GymID, domain.StaffRoles).Error; err != nil {
			respondError(w, http.StatusNotFound, "Pessoa não encontrada na equipe")
			return
		}
		who = "de " + u.Name
		q = q.Where("user_id = ?", u.ID)
	} else {
		req.UserID = nil
		q = q.Where("user_id IS NULL")
	}
	var g domain.Goal
	exists := q.First(&g).Error == nil
	if req.TargetSales == 0 && req.TargetRevenue == 0 {
		if exists {
			h.db.Delete(&g)
			audit(h.db, r, "goal.delete", "goal", g.ID, "Removeu a meta %s em %s", who, month.Format("01/2006"))
		}
		respondJSON(w, http.StatusOK, map[string]bool{"success": true})
		return
	}
	g.GymID, g.UserID, g.Month, g.TargetSales, g.TargetRevenue = claims.GymID, req.UserID, ref, req.TargetSales, req.TargetRevenue
	if exists {
		err = h.db.Save(&g).Error
	} else {
		err = h.db.Create(&g).Error
	}
	if err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "goal.set", "goal", g.ID, "Definiu a meta %s em %s: %d vendas / R$ %s", who, month.Format("01/2006"), g.TargetSales, service.FormatBRL(g.TargetRevenue))
	respondJSON(w, http.StatusOK, g)
}
