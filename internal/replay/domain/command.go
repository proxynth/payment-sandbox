package domain

import (
	"fmt"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

type CommandType string

const (
	CommandCreatePayment CommandType = "create_payment"
	CommandAuthorize     CommandType = "authorize"
	CommandCapture       CommandType = "capture"
	CommandRefund        CommandType = "refund"
	CommandCancel        CommandType = "cancel"
)

func (commandType CommandType) Valid() bool {
	switch commandType {
	case CommandCreatePayment, CommandAuthorize, CommandCapture, CommandRefund, CommandCancel:
		return true
	default:
		return false
	}
}

// Command represents one ordered business operation in a scenario. Amount is
// required for create, capture, and refund, and must be empty otherwise.
type Command struct {
	Type      CommandType
	PaymentID paymentdomain.ID
	Amount    paymentdomain.Money
}

func (command Command) Validate() error {
	if !command.Type.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidCommandType, command.Type)
	}

	if command.PaymentID == "" {
		return ErrInvalidCommandPaymentID
	}

	switch command.Type {
	case CommandCreatePayment, CommandCapture, CommandRefund:
		if command.Amount.Amount() <= 0 {
			return ErrInvalidCommandAmount
		}

		if _, err := paymentdomain.NewMoney(command.Amount.Amount(), command.Amount.Currency()); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidCommandAmount, err)
		}
	case CommandAuthorize, CommandCancel:
		if command.Amount.Amount() != 0 || command.Amount.Currency() != "" {
			return ErrInvalidCommandAmount
		}
	}

	return nil
}
