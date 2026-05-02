package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/DotNaos/moodle-services/pkg/chatgptapp"
)

func PDF(w http.ResponseWriter, r *http.Request) {
	setPDFCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := chatgptapp.LoadConfigFromEnv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	service, _, status, err := serviceFromRequest(r, cfg)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	courseID := strings.TrimSpace(r.URL.Query().Get("courseId"))
	resourceID := strings.TrimSpace(r.URL.Query().Get("resourceId"))
	if courseID == "" || resourceID == "" {
		http.Error(w, "courseId and resourceId are required", http.StatusBadRequest)
		return
	}

	file, err := service.PDFFile(courseID, resourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", safeFilename(file.Descriptor.Title)+".pdf"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Data)
}

func setPDFCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "authorization, content-type, x-moodle-app-key")
}

func safeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "moodle-document"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\x00", "")
	return replacer.Replace(value)
}
