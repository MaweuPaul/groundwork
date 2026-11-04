package handler

import (
	"log"
	"net/http"

	"github.com/google/uuid"

	"groundwork/internal/httpx"
)

// resolveRef turns the URL's {id} (a UUID or a short ID) into a UUID. When it names
// nothing it writes notFound as a 404, on a database error a 500, and returns false.
func resolveRef(w http.ResponseWriter, r *http.Request, resolve func(string) (uuid.UUID, bool, error), notFound string) (uuid.UUID, bool) {
	ref := r.PathValue("id")
	id, found, err := resolve(ref)
	if err != nil {
		log.Printf("resolve %q: %v", ref, err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return uuid.Nil, false
	}
	if !found {
		httpx.WriteError(w, http.StatusNotFound, notFound)
		return uuid.Nil, false
	}
	return id, true
}
