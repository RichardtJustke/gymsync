package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"gymsync/backend/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

// respondErr devolve a mensagem de regras de negócio como 400; o resto vira 500 genérico (detalhe só no log)
func respondErr(w http.ResponseWriter, err error) {
	if service.IsBusiness(err) {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("[ERRO] %v", err)
	respondError(w, http.StatusInternalServerError, "Erro interno, tente novamente")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "Requisição muito grande")
		} else {
			respondError(w, http.StatusBadRequest, "JSON inválido")
		}
		return false
	}
	return true
}

func urlUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		respondError(w, http.StatusBadRequest, "ID inválido")
		return uuid.Nil, false
	}
	return id, true
}

func cleanDocument(doc string) string { return service.CleanDocument(doc) }

func chiParam(r *http.Request, name string) string { return chi.URLParam(r, name) }
