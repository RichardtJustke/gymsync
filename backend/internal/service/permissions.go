package service

import (
	"strings"
	"sync"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Permissions resolve o que cada cargo pode fazer em cada academia (com cache em memória)
type Permissions struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache map[string]map[domain.Permission]bool
}

func NewPermissions(db *gorm.DB) *Permissions {
	return &Permissions{db: db, cache: map[string]map[domain.Permission]bool{}}
}

func permKey(gymID uuid.UUID, role domain.Role) string { return gymID.String() + "|" + string(role) }

// For devolve o conjunto de permissões do cargo (dono tem todas; aluno nenhuma)
func (p *Permissions) For(gymID uuid.UUID, role domain.Role) map[domain.Permission]bool {
	switch role {
	case domain.RoleAdmin:
		return setOf(domain.AllPermissions())
	case domain.RoleAluno, "":
		return map[domain.Permission]bool{}
	}
	key := permKey(gymID, role)
	p.mu.RLock()
	cached, ok := p.cache[key]
	p.mu.RUnlock()
	if ok {
		return cached
	}

	var row domain.RolePermission
	var set map[domain.Permission]bool
	if p.db.Where("gym_id = ? AND role = ?", gymID, role).First(&row).Error == nil {
		set = setOf(ParsePermissions(row.Permissions))
	} else {
		set = setOf(domain.DefaultPermissions[role])
	}
	p.mu.Lock()
	p.cache[key] = set
	p.mu.Unlock()
	return set
}

// Has indica se o cargo tem alguma das permissões
func (p *Permissions) Has(gymID uuid.UUID, role domain.Role, perms ...domain.Permission) bool {
	set := p.For(gymID, role)
	for _, perm := range perms {
		if set[perm] {
			return true
		}
	}
	return false
}

// List permissões do cargo na ordem do catálogo
func (p *Permissions) List(gymID uuid.UUID, role domain.Role) []domain.Permission {
	set := p.For(gymID, role)
	out := []domain.Permission{}
	for _, info := range domain.PermissionCatalog {
		if set[info.Key] {
			out = append(out, info.Key)
		}
	}
	return out
}

// Set grava as permissões de um cargo (o dono não é configurável)
func (p *Permissions) Set(gymID uuid.UUID, role domain.Role, perms []domain.Permission) error {
	if role == domain.RoleAdmin || !domain.IsStaffRole(role) {
		return bizErr("Este cargo não tem permissões configuráveis")
	}
	valid := setOf(domain.AllPermissions())
	clean := []string{}
	seen := map[domain.Permission]bool{}
	for _, perm := range perms {
		if valid[perm] && !seen[perm] {
			seen[perm] = true
			clean = append(clean, string(perm))
		}
	}
	var row domain.RolePermission
	err := p.db.Where("gym_id = ? AND role = ?", gymID, role).First(&row).Error
	row.GymID, row.Role, row.Permissions = gymID, role, strings.Join(clean, ",")
	if err != nil {
		err = p.db.Create(&row).Error
	} else {
		err = p.db.Save(&row).Error
	}
	if err != nil {
		return err
	}
	p.mu.Lock()
	delete(p.cache, permKey(gymID, role))
	p.mu.Unlock()
	return nil
}

// Reset volta o cargo para as permissões padrão
func (p *Permissions) Reset(gymID uuid.UUID, role domain.Role) error {
	if err := p.db.Where("gym_id = ? AND role = ?", gymID, role).Delete(&domain.RolePermission{}).Error; err != nil {
		return err
	}
	p.mu.Lock()
	delete(p.cache, permKey(gymID, role))
	p.mu.Unlock()
	return nil
}

// UsersWith ids da equipe ativa com a permissão (para notificar quem cuida daquela área)
func (p *Permissions) UsersWith(gymID uuid.UUID, perm domain.Permission) []uuid.UUID {
	roles := []domain.Role{}
	for _, r := range domain.StaffRoles {
		if p.Has(gymID, r, perm) {
			roles = append(roles, r)
		}
	}
	var ids []uuid.UUID
	if len(roles) > 0 {
		p.db.Model(&domain.User{}).Where("gym_id = ? AND role IN ? AND active = ?", gymID, roles, true).Pluck("id", &ids)
	}
	return ids
}

// ParsePermissions lê a lista separada por vírgula
func ParsePermissions(s string) []domain.Permission {
	out := []domain.Permission{}
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, domain.Permission(part))
		}
	}
	return out
}

func setOf(perms []domain.Permission) map[domain.Permission]bool {
	m := make(map[domain.Permission]bool, len(perms))
	for _, p := range perms {
		m[p] = true
	}
	return m
}
