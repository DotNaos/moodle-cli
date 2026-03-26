package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	LastUpdateCheckAt time.Time `json:"lastUpdateCheckAt,omitempty"`
	LastNotifiedTag   string    `json:"lastNotifiedTag,omitempty"`
}

func LoadState(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func SaveState(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func ShouldCheck(state State, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return true
	}
	if state.LastUpdateCheckAt.IsZero() {
		return true
	}
	return now.Sub(state.LastUpdateCheckAt) >= interval
}
