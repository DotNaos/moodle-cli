package moodle

import (
	"errors"
	"testing"
)

func TestIsMissingPlaywrightDriverError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "missing driver message",
			err:  errors.New("please install the driver (v1.52.0) first"),
			want: true,
		},
		{
			name: "mixed case message",
			err:  errors.New("Please Install The Driver (v1.52.0) first"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("network unavailable"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMissingPlaywrightDriverError(tt.err)
			if got != tt.want {
				t.Fatalf("isMissingPlaywrightDriverError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestResolveSchoolRejectsInactiveSchool(t *testing.T) {
	_, err := resolveSchool("phgr")
	if err == nil {
		t.Fatal("expected inactive school to be rejected")
	}
	if err.Error() != `school id "phgr" is not active; multi-school support is not active` {
		t.Fatalf("unexpected error: %v", err)
	}
}
