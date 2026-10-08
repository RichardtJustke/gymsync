package handler

import (
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"gorm.io/gorm"
)

// EquipmentHandler aparelhos e manutenção
type EquipmentHandler struct {
	db *gorm.DB
}

func NewEquipmentHandler(db *gorm.DB) *EquipmentHandler {
	return &EquipmentHandler{db: db}
}

func (h *EquipmentHandler) List(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var list []domain.Equipment
	h.db.Where("gym_id = ?", gymID).Order("sector asc, name asc").Find(&list)
	respondJSON(w, http.StatusOK, list)
}

type EquipmentRequest struct {
	Name            string                 `json:"name"`
	Sector          string                 `json:"sector"`
	Status          domain.EquipmentStatus `json:"status"`
	Notes           string                 `json:"notes"`
	LastMaintenance string                 `json:"last_maintenance"` // AAAA-MM-DD
	NextMaintenance string                 `json:"next_maintenance"`
}

func parseOptDate(s string) (*time.Time, bool) {
	if s == "" {
		return nil, true
	}
	t, err := time.ParseInLocation("2006-01-02", s, config.Loc())
	if err != nil {
		return nil, false
	}
	return &t, true
}

func (req EquipmentRequest) apply(e *domain.Equipment) string {
	if strings.TrimSpace(req.Name) == "" {
		return "Nome do equipamento é obrigatório"
	}
	switch req.Status {
	case domain.EquipOK, domain.EquipWarning, domain.EquipMaintenance:
		e.Status = req.Status
	case "":
		e.Status = domain.EquipOK
	default:
		return "Status inválido"
	}
	last, ok1 := parseOptDate(req.LastMaintenance)
	next, ok2 := parseOptDate(req.NextMaintenance)
	if !ok1 || !ok2 {
		return "Data inválida"
	}
	e.Name, e.Sector, e.Notes = strings.TrimSpace(req.Name), strings.TrimSpace(req.Sector), req.Notes
	e.LastMaintenance, e.NextMaintenance = last, next
	return ""
}

func (h *EquipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req EquipmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e := domain.Equipment{GymID: gymID}
	if msg := req.apply(&e); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}
	if err := h.db.Create(&e).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "equipment.create", "equipment", e.ID, "Cadastrou o equipamento %s", e.Name)
	respondJSON(w, http.StatusCreated, e)
}

func (h *EquipmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var e domain.Equipment
	if err := h.db.First(&e, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Equipamento não encontrado")
		return
	}
	var req EquipmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	old := e.Status
	if msg := req.apply(&e); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}
	if err := h.db.Save(&e).Error; err != nil {
		respondErr(w, err)
		return
	}
	if old != e.Status {
		audit(h.db, r, "equipment.status", "equipment", e.ID, "%s: %s → %s", e.Name, old, e.Status)
	} else {
		audit(h.db, r, "equipment.update", "equipment", e.ID, "Atualizou o equipamento %s", e.Name)
	}
	respondJSON(w, http.StatusOK, e)
}

func (h *EquipmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var e domain.Equipment
	if err := h.db.First(&e, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Equipamento não encontrado")
		return
	}
	h.db.Delete(&e)
	audit(h.db, r, "equipment.delete", "equipment", e.ID, "Removeu o equipamento %s", e.Name)
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}
