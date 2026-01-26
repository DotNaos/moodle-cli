package config

import (
  "encoding/json"
  "os"
  "path/filepath"
)

type Config struct {
  SchoolID    string `json:"schoolId,omitempty"`
  Username    string `json:"username,omitempty"`
  Password    string `json:"password,omitempty"`
  CalendarURL string `json:"calendarUrl,omitempty"`
}

func LoadConfig(path string) (Config, error) {
  data, err := os.ReadFile(path)
  if err != nil {
    if os.IsNotExist(err) {
      return Config{}, nil
    }
    return Config{}, err
  }
  var cfg Config
  if err := json.Unmarshal(data, &cfg); err != nil {
    return Config{}, err
  }
  return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
  if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
    return err
  }
  data, err := json.MarshalIndent(cfg, "", "  ")
  if err != nil {
    return err
  }
  return os.WriteFile(path, data, 0o600)
}
