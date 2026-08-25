package domain

import (
	"fmt"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

type CommandType string

const (
	CommandCreatePayment CommandType = "create_payment"
	CommandStartSaga     CommandType = "start_payment_saga"
	CommandAuthorize     CommandType = "authorize"
	CommandCapture       CommandType = "capture"
	CommandRefund        CommandType = "refund"
	CommandCancel        CommandType = "cancel"
	CommandAdvanceTime   CommandType = "advance_time"
	CommandExecuteAsync  CommandType = "execute_async"
)

func (commandType CommandType) Valid() bool {
	switch commandType {
	case CommandCreatePayment, CommandStartSaga, CommandAuthorize, CommandCapture, CommandRefund, CommandCancel, CommandAdvanceTime, CommandExecuteAsync:
		return true
	default:
		return false
	}
}

// Command represents one ordered business operation in a scenario. Amount is
// required for create, capture, and refund, and must be empty otherwise.
type Command struct {
	Type        CommandType
	PaymentID   paymentdomain.ID
	Amount      paymentdomain.Money
	Duration    time.Duration
	OperationID string
}

func (command Command) Validate() error {
	if !command.Type.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidCommandType, command.Type)
	}

	if command.PaymentID == "" && command.Type != CommandAdvanceTime && command.Type != CommandExecuteAsync {
		return ErrInvalidCommandPaymentID
	}

	switch command.Type {
	case CommandCreatePayment, CommandStartSaga, CommandCapture, CommandRefund:
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
	case CommandAdvanceTime:
		if command.PaymentID != "" || command.Amount.Amount() != 0 || command.Amount.Currency() != "" || command.Duration <= 0 {
			return ErrInvalidCommandAmount
		}
	case CommandExecuteAsync:
		if command.PaymentID != "" || command.Amount.Amount() != 0 || command.Amount.Currency() != "" || command.OperationID == "" {
			return ErrInvalidCommandPaymentID
		}
	}

	return nil
}
