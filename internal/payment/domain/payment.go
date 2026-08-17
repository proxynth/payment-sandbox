package domain

type ID string

type Status string

func (s Status) Valid() bool {
	switch s {
	case StatusPending:
		return true
	default:
		return false
	}
}

const (
	StatusPending Status = "pending"
)

type Payment struct {
	id      ID
	status  Status
	version uint64
}

func New(id ID) (*Payment, error) {
	if id == "" {
		return nil, ErrInvalidPaymentID
	}

	return &Payment{
		id:      id,
		status:  StatusPending,
		version: 1,
	}, nil
}

func Restore(
	id ID,
	status Status,
	version uint64,
) (*Payment, error) {
	if id == "" {
		return nil, ErrInvalidPaymentID
	}

	if !status.Valid() {
		return nil, ErrInvalidPaymentStatus
	}

	if version == 0 {
		return nil, ErrInvalidPaymentVersion
	}

	return &Payment{
		id:      id,
		status:  status,
		version: version,
	}, nil
}

func (p *Payment) ID() ID {
	return p.id
}

func (p *Payment) Status() Status {
	return p.status
}

func (p *Payment) Version() uint64 {
	return p.version
}
