package domain

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	payment, err := New("payment-1")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if payment.ID() != "payment-1" {
		t.Fatalf("ID() = %q, want %q", payment.ID(), "payment-1")
	}

	if payment.Status() != StatusPending {
		t.Fatalf("Status() = %q, want %q", payment.Status(), StatusPending)
	}

	if payment.Version() != 1 {
		t.Fatalf("Version() = %d, want %d", payment.Version(), 1)
	}
}

func TestNew_RejectsEmptyID(t *testing.T) {
	_, err := New("")

	if !errors.Is(err, ErrInvalidPaymentID) {
		t.Fatalf("New() error = %v, want %v", err, ErrInvalidPaymentID)
	}
}

func TestRestore(t *testing.T) {
	payment, err := Restore(
		"payment-1",
		StatusPending,
		4,
	)

	if err != nil {
		t.Fatalf("failed to restore payment: %v", err)
	}

	if payment.ID() != "payment-1" {
		t.Fatalf("payment id should be 'payment-1'")
	}

	if payment.Status() != StatusPending {
		t.Fatalf("payment status should be '%s'", StatusPending)
	}

	if payment.Version() != 4 {
		t.Fatalf("payment version should be 4")
	}
}

func TestRestore_RejectsEmptyID(t *testing.T) {
	_, err := Restore(
		"",
		StatusPending,
		1,
	)

	if !errors.Is(err, ErrInvalidPaymentID) {
		t.Fatalf("error should be 'ErrInvalidPaymentID'")
	}
}

func TestRestore_RejectsInvalidStatus(t *testing.T) {
	_, err := Restore(
		"payment-1",
		Status("foobar"),
		1,
	)

	if !errors.Is(err, ErrInvalidPaymentStatus) {
		t.Fatalf("error should be 'ErrInvalidPaymentStatus'")
	}
}

func TestRestore_RejectsInvalidVersion(t *testing.T) {
	_, err := Restore(
		"payment-1",
		StatusPending,
		0,
	)

	if !errors.Is(err, ErrInvalidPaymentVersion) {
		t.Fatalf("error should be 'ErrInvalidPaymentVersion'")
	}
}

func TestStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "pending",
			status: StatusPending,
			want:   true,
		},
		{
			name:   "unknown",
			status: Status("foobar"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.Valid(); got != tt.want {
				t.Errorf("Status.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
