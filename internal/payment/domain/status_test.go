package domain

import "testing"

func TestStatus_Terminal(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPending, false},
		{StatusAuthorized, false},
		{StatusPartiallyCaptured, false},
		{StatusCaptured, false},
		{StatusPartiallyRefunded, false},
		{StatusRefunded, true},
		{StatusFailed, true},
		{StatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.Terminal(); got != tt.want {
				t.Errorf(
					"Terminal() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}
