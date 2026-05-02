package handler

import (
	"net/http"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

func Courses(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodGet) {
		return
	}
	service, closeFn, err := svc.ServiceForRequest(r, svc.LoadServerEnv())
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	defer closeFn()
	courses, err := service.ListCourses()
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	svc.WriteJSON(w, http.StatusOK, map[string]any{"courses": courses})
}
