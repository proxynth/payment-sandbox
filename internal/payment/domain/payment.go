package domain

import "fmt"

type ID string

type Payment struct {
	id               ID
	amount           Money
	status           Status
	authorizedAmount int64
	capturedAmount   int64
	refundedAmount   int64
	version          uint64
}

func New(id ID, amount Money) (*Payment, error) {
	if id == "" {
		return nil, ErrInvalidPaymentID
	}

	if amount.Amount() <= 0 {
		return nil, ErrInvalidMoneyAmount
	}

	return &Payment{
		id:      id,
		amount:  amount,
		status:  StatusPending,
		version: 1,
	}, nil
}

func Restore(
	state PaymentState,
) (*Payment, error) {

	if err := state.Validate(); err != nil {
		return nil, err
	}

	return &Payment{
		id:               state.ID,
		amount:           state.Amount,
		status:           state.Status,
		authorizedAmount: state.AuthorizedAmount,
		capturedAmount:   state.CapturedAmount,
		refundedAmount:   state.RefundedAmount,
		version:          state.Version,
	}, nil
}

func (p *Payment) ID() ID {
	return p.id
}

func (p *Payment) Status() Status {
	return p.status
}

func (p *Payment) Amount() Money {
	return p.amount
}

func (p *Payment) AuthorizedAmount() Money {
	return Money{
		amount:   p.authorizedAmount,
		currency: p.amount.currency,
	}
}

func (p *Payment) CapturedAmount() Money {
	return Money{
		amount:   p.capturedAmount,
		currency: p.amount.currency,
	}
}

func (p *Payment) RefundedAmount() Money {
	return Money{
		amount:   p.refundedAmount,
		currency: p.amount.currency,
	}
}

func (p *Payment) RemainingCapturableAmount() Money {
	return Money{
		amount:   p.authorizedAmount - p.capturedAmount,
		currency: p.amount.currency,
	}
}

func (p *Payment) RemainingRefundableAmount() Money {
	return Money{
		amount:   p.capturedAmount - p.refundedAmount,
		currency: p.amount.currency,
	}
}

func (p *Payment) Version() uint64 {
	return p.version
}

func (p *Payment) Authorize() error {
	if p.status != StatusPending {
		return fmt.Errorf(
			"%w: authorize payment in status %q",
			ErrInvalidTransition,
			p.status,
		)
	}

	p.authorizedAmount = p.amount.Amount()
	p.status = StatusAuthorized
	p.version++

	return nil
}

func (p *Payment) Fail() error {
	if p.status != StatusPending {
		return fmt.Errorf(
			"%w: fail payment in status %q",
			ErrInvalidTransition,
			p.status,
		)
	}

	p.status = StatusFailed
	p.version++

	return nil
}

func (p *Payment) Cancel() error {
	if p.status != StatusAuthorized {
		return fmt.Errorf(
			"%w: cancel payment in status %q",
			ErrInvalidTransition,
			p.status,
		)
	}

	p.status = StatusCancelled
	p.version++

	return nil
}

func (p *Payment) Capture(amount Money) error {
	if p.status != StatusAuthorized && p.status != StatusPartiallyCaptured {
		return fmt.Errorf(
			"%w: capture payment in status %q",
			ErrInvalidTransition,
			p.status,
		)
	}

	if amount.Currency() != p.amount.currency {
		return ErrInvalidCurrency
	}

	if amount.Amount() <= 0 {
		return ErrInvalidCapturedAmount
	}

	remaining := p.authorizedAmount - p.capturedAmount

	if amount.Amount() > remaining {
		return ErrInvalidCapturedAmount
	}

	p.capturedAmount += amount.Amount()

	if p.capturedAmount == p.authorizedAmount {
		p.status = StatusCaptured
	} else {
		p.status = StatusPartiallyCaptured
	}

	p.version++

	return nil
}

func (p *Payment) Refund(amount Money) error {
	if p.status != StatusCaptured && p.status != StatusPartiallyRefunded {
		return fmt.Errorf(
			"%w: refund payment in status %q",
			ErrInvalidTransition,
			p.status,
		)
	}

	if amount.Currency() != p.amount.currency {
		return ErrInvalidCurrency
	}

	if amount.Amount() <= 0 {
		return ErrInvalidRefundedAmount
	}

	remaining := p.capturedAmount - p.refundedAmount

	if amount.Amount() > remaining {
		return ErrInvalidRefundedAmount
	}

	p.refundedAmount += amount.Amount()

	if p.refundedAmount == p.capturedAmount {
		p.status = StatusRefunded
	} else {
		p.status = StatusPartiallyRefunded
	}

	p.version++

	return nil
}
