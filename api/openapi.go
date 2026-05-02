package handler

import (
	"net/http"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

func Openapi(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodGet) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(svc.OpenAPISpecJSON())
}
