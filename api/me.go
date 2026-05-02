package handler

import (
	"net/http"

	svc "github.com/DotNaos/moodle-services/pkg/moodleservices"
)

func Me(w http.ResponseWriter, r *http.Request) {
	if !svc.AllowMethods(w, r, http.MethodGet) {
		return
	}
	st, user, _, err := svc.AuthenticatedUser(r, svc.LoadServerEnv())
	if err != nil {
		svc.WriteError(w, err)
		return
	}
	defer st.Close()
	svc.WriteJSON(w, http.StatusOK, user)
}
