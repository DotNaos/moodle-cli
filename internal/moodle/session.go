package moodle

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "time"
)

type Session struct {
  SchoolID  string    `json:"schoolId"`
  Cookies   string    `json:"cookies"`
  CreatedAt time.Time `json:"createdAt"`
}

func LoadSession(path string) (Session, error) {
  data, err := os.ReadFile(path)
  if err != nil {
    return Session{}, err
  }
  var s Session
  if err := json.Unmarshal(data, &s); err != nil {
    return Session{}, err
  }
  if s.Cookies == "" {
    return Session{}, fmt.Errorf("session file missing cookies")
  }
  if s.SchoolID == "" {
    return Session{}, fmt.Errorf("session file missing schoolId")
  }
  return s, nil
}

func SaveSession(path string, s Session) error {
  if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
    return err
  }
  data, err := json.MarshalIndent(s, "", "  ")
  if err != nil {
    return err
  }
  return os.WriteFile(path, data, 0o600)
}
