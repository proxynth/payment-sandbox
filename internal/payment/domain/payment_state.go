package domain

type Status string

type PaymentState struct {
	ID               ID
	Amount           Money
	Status           Status
	AuthorizedAmount int64
	CapturedAmount   int64
	RefundedAmount   int64
	Version          uint64
}

const (
	StatusPending           Status = "pending"
	StatusAuthorized        Status = "authorized"
	StatusPartiallyCaptured Status = "partially_captured"
	StatusCaptured          Status = "captured"
	StatusPartiallyRefunded Status = "partially_refunded"
	StatusRefunded          Status = "refunded"
	StatusFailed            Status = "failed"
	StatusCancelled         Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending,
		StatusAuthorized,
		StatusPartiallyCaptured,
		StatusCaptured,
		StatusPartiallyRefunded,
		StatusRefunded,
		StatusFailed,
		StatusCancelled:
		return true
	default:
		return false
	}
}

func (s Status) Terminal() bool {
	switch s {
	case StatusFailed,
		StatusCancelled,
		StatusRefunded:
		return true
	default:
		return false
	}
}

func (s PaymentState) Validate() error {
	if s.ID == "" {
		return ErrInvalidPaymentID
	}

	if !s.Status.Valid() {
		return ErrInvalidPaymentStatus
	}

	if s.Version == 0 {
		return ErrInvalidPaymentVersion
	}

	if s.Amount.Amount() <= 0 {
		return ErrInvalidMoneyAmount
	}

	if s.AuthorizedAmount < 0 || s.AuthorizedAmount > s.Amount.Amount() {
		return ErrInvalidAuthorizedAmount
	}

	if s.CapturedAmount < 0 || s.CapturedAmount > s.AuthorizedAmount {
		return ErrInvalidCapturedAmount
	}

	if s.RefundedAmount < 0 || s.RefundedAmount > s.CapturedAmount {
		return ErrInvalidRefundedAmount
	}

	return s.validateLifecycleConsistency()
}

func (s PaymentState) validateLifecycleConsistency() error {
	switch s.Status {
	case StatusPending, StatusFailed:
		if s.AuthorizedAmount != 0 ||
			s.CapturedAmount != 0 ||
			s.RefundedAmount != 0 {
			return ErrInvalidPaymentState
		}
	case StatusAuthorized:
		if s.AuthorizedAmount != s.Amount.Amount() ||
			s.CapturedAmount != 0 ||
			s.RefundedAmount != 0 {
			return ErrInvalidPaymentState
		}
	case StatusPartiallyCaptured:
		if s.AuthorizedAmount != s.Amount.Amount() ||
			s.CapturedAmount <= 0 ||
			s.CapturedAmount >= s.AuthorizedAmount ||
			s.RefundedAmount != 0 {
			return ErrInvalidPaymentState
		}
	case StatusCaptured:
		if s.AuthorizedAmount != s.Amount.Amount() ||
			s.CapturedAmount != s.AuthorizedAmount ||
			s.RefundedAmount != 0 {
			return ErrInvalidPaymentState
		}
	case StatusPartiallyRefunded:
		if s.AuthorizedAmount != s.Amount.Amount() ||
			s.CapturedAmount != s.AuthorizedAmount ||
			s.RefundedAmount <= 0 ||
			s.RefundedAmount >= s.CapturedAmount {
			return ErrInvalidPaymentState
		}
	case StatusRefunded:
		if s.AuthorizedAmount != s.Amount.Amount() ||
			s.CapturedAmount != s.AuthorizedAmount ||
			s.RefundedAmount != s.CapturedAmount {
			return ErrInvalidPaymentState
		}
	case StatusCancelled:
		if s.AuthorizedAmount != s.Amount.Amount() ||
			s.CapturedAmount != 0 ||
			s.RefundedAmount != 0 {
			return ErrInvalidPaymentState
		}
	}

	return nil
}
