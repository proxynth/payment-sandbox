package application

import (
	"context"
	"encoding/json"
	"fmt"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	sagadoamin "proxynth/payment-sandbox/internal/saga/domain"
)

// PaymentExecutor is the application boundary between the Saga and the
// payment/provider contexts. It never changes payment state without going
// through the payment application services.
type PaymentExecutor struct {
	payments  paymentapplication.Repository
	publisher paymentapplication.EventPublisher
	provider  providerdomain.Provider
	clock     clock.Clock
}

func NewPaymentExecutor(payments paymentapplication.Repository, provider providerdomain.Provider, businessClock clock.Clock) (*PaymentExecutor, error) {
	return NewPaymentExecutorWithPublisher(payments, provider, businessClock, nil)
}

func NewPaymentExecutorWithPublisher(payments paymentapplication.Repository, provider providerdomain.Provider, businessClock clock.Clock, publisher paymentapplication.EventPublisher) (*PaymentExecutor, error) {
	if payments == nil || provider == nil || businessClock == nil {
		return nil, fmt.Errorf("invalid payment saga executor")
	}
	return &PaymentExecutor{payments: payments, publisher: publisher, provider: provider, clock: businessClock}, nil
}

func (e *PaymentExecutor) Execute(ctx context.Context, message sagadoamin.Message) (Execution, error) {
	payment, err := e.payments.FindByID(ctx, message.PaymentID)
	if err != nil {
		return Execution{}, err
	}

	var input struct {
		Amount   int64                  `json:"amount"`
		Currency paymentdomain.Currency `json:"currency"`
	}
	if len(message.Payload) > 0 {
		if err := json.Unmarshal(message.Payload, &input); err != nil {
			return Execution{}, err
		}
	}

	snapshot := providerdomain.PaymentSnapshot{
		ID: payment.ID(), Amount: payment.Amount(), Status: payment.Status(), Version: payment.Version(),
	}
	var result providerdomain.OperationResult
	if message.OperationID != "" {
		asyncProvider, ok := e.provider.(providerdomain.AsyncExecutor)
		if !ok {
			return Execution{}, fmt.Errorf("provider does not support async execution")
		}
		var operation providerdomain.AsyncOperation
		if err := json.Unmarshal(message.Payload, &operation); err != nil {
			return Execution{}, err
		}
		result, err = asyncProvider.ExecuteAsync(ctx, operation)
	} else {
		switch message.Step {
		case sagadoamin.StepAuthorize:
			result, err = e.provider.Authorize(ctx, providerdomain.AuthorizeRequest{Payment: snapshot, At: e.clock.Now()})
		case sagadoamin.StepCapture:
			amount, moneyErr := paymentdomain.NewMoney(input.Amount, input.Currency)
			if moneyErr != nil {
				return Execution{}, moneyErr
			}
			result, err = e.provider.Capture(ctx, providerdomain.CaptureRequest{Payment: snapshot, Amount: amount, At: e.clock.Now()})
		case sagadoamin.StepRefund:
			amount, moneyErr := paymentdomain.NewMoney(input.Amount, input.Currency)
			if moneyErr != nil {
				return Execution{}, moneyErr
			}
			result, err = e.provider.Refund(ctx, providerdomain.RefundRequest{Payment: snapshot, Amount: amount, At: e.clock.Now()})
		case sagadoamin.StepCancel:
			result, err = e.provider.Cancel(ctx, providerdomain.CancelRequest{Payment: snapshot, At: e.clock.Now()})
		default:
			return Execution{}, sagadoamin.ErrInvalidStep
		}
	}
	if err != nil {
		return Execution{}, err
	}
	if err := result.Validate(); err != nil {
		return Execution{}, err
	}
	if result.Outcome == providerdomain.OutcomePending {
		return Execution{Outcome: OutcomePending, AsyncOperations: result.AsyncOperations}, nil
	}
	if result.Outcome == providerdomain.OutcomeFailed {
		if message.Step == sagadoamin.StepAuthorize {
			if _, err := paymentapplication.NewFailPaymentWithPublisher(e.payments, e.publisher).Execute(ctx, paymentapplication.FailPaymentCommand{PaymentID: message.PaymentID}); err != nil {
				return Execution{}, err
			}
		}
		return Execution{Outcome: OutcomeFailed}, nil
	}

	switch message.Step {
	case sagadoamin.StepAuthorize:
		_, err = paymentapplication.NewAuthorizePaymentWithPublisher(e.payments, e.publisher).Execute(ctx, paymentapplication.AuthorizePaymentCommand{PaymentID: message.PaymentID})
	case sagadoamin.StepCapture:
		_, err = paymentapplication.NewCapturePaymentWithPublisher(e.payments, e.publisher).Execute(ctx, paymentapplication.CapturePaymentCommand{PaymentID: message.PaymentID, Amount: input.Amount, Currency: input.Currency})
	case sagadoamin.StepRefund:
		_, err = paymentapplication.NewRefundPaymentWithPublisher(e.payments, e.publisher).Execute(ctx, paymentapplication.RefundPaymentCommand{PaymentID: message.PaymentID, Amount: input.Amount, Currency: input.Currency})
	case sagadoamin.StepCancel:
		_, err = paymentapplication.NewCancelPaymentWithPublisher(e.payments, e.publisher).Execute(ctx, paymentapplication.CancelPaymentCommand{PaymentID: message.PaymentID})
	}
	if err != nil {
		return Execution{}, err
	}
	return Execution{Outcome: OutcomeSucceeded}, nil
}
