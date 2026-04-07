package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DotNaos/moodle-cli/internal/moodle"
)

type stubClient struct {
	validateErr error
	courses     []moodle.Course
	resources   map[string][]moodle.Resource
}

func (s stubClient) ValidateSession() error {
	return s.validateErr
}

func (s stubClient) FetchCourses() ([]moodle.Course, error) {
	return s.courses, nil
}

func (s stubClient) FetchCourseResources(courseID string) ([]moodle.Resource, string, error) {
	if s.resources == nil {
		return nil, "", fmt.Errorf("no resources configured")
	}
	res, ok := s.resources[courseID]
	if !ok {
		return nil, "", fmt.Errorf("course %s not found", courseID)
	}
	return res, "", nil
}

func TestHealthHandlerOK(t *testing.T) {
	router, err := NewRouter(ServerOptions{
		ClientProvider: func() (Client, error) {
			return stubClient{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestHealthHandlerExpiredSession(t *testing.T) {
	router, err := NewRouter(ServerOptions{
		ClientProvider: func() (Client, error) {
			return stubClient{validateErr: moodle.ErrSessionExpired}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCoursesHandler(t *testing.T) {
	wantCourses := []moodle.Course{
		{ID: 1, Fullname: "Course A", Category: "Cat"},
	}
	router, err := NewRouter(ServerOptions{
		ClientProvider: func() (Client, error) {
			return stubClient{courses: wantCourses}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var got []moodle.Course
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != len(wantCourses) || got[0].ID != wantCourses[0].ID {
		t.Fatalf("unexpected courses: %#v", got)
	}
}

func TestCourseResourcesHandler(t *testing.T) {
	resource := moodle.Resource{ID: "42", Name: "Slide"}
	router, err := NewRouter(ServerOptions{
		ClientProvider: func() (Client, error) {
			return stubClient{
				resources: map[string][]moodle.Resource{
					"123": {resource},
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/courses/123/resources", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var got []moodle.Resource
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 || got[0].ID != resource.ID {
		t.Fatalf("unexpected resources: %#v", got)
	}
}
