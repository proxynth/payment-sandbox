package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"proxynth/payment-sandbox/internal/api"
	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/observability"
)

const paymentPathPrefix = "/payments/"

type Handler struct {
	create    *paymentapplication.CreatePayment
	get       *paymentapplication.GetPayment
	authorize *paymentapplication.AuthorizePayment
	capture   *paymentapplication.CapturePayment
	refund    *paymentapplication.RefundPayment
	cancel    *paymentapplication.CancelPayment
}

func NewHandler(repository paymentapplication.Repository) (*Handler, error) {
	return NewHandlerWithPublisher(repository, nil)
}

func NewHandlerWithPublisher(repository paymentapplication.Repository, publisher paymentapplication.EventPublisher) (*Handler, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &Handler{
		create:    paymentapplication.NewCreatePaymentWithPublisher(repository, publisher),
		get:       paymentapplication.NewGetPayment(repository),
		authorize: paymentapplication.NewAuthorizePaymentWithPublisher(repository, publisher),
		capture:   paymentapplication.NewCapturePaymentWithPublisher(repository, publisher),
		refund:    paymentapplication.NewRefundPaymentWithPublisher(repository, publisher),
		cancel:    paymentapplication.NewCancelPaymentWithPublisher(repository, publisher),
	}, nil
}

func (h *Handler) Register(server *api.Server) error {
	if err := server.Handle(http.MethodPost, "/payments", http.HandlerFunc(h.createPayment)); err != nil {
		return err
	}
	if err := server.HandlePrefix(http.MethodGet, paymentPathPrefix, http.HandlerFunc(h.getPayment)); err != nil {
		return err
	}
	return server.HandlePrefix(http.MethodPost, paymentPathPrefix, http.HandlerFunc(h.command))
}

type createPaymentRequest struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type amountRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type paymentResponse struct {
	ID               string `json:"id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	AuthorizedAmount int64  `json:"authorized_amount"`
	CapturedAmount   int64  `json:"captured_amount"`
	RefundedAmount   int64  `json:"refunded_amount"`
	Version          uint64 `json:"version"`
}

func (h *Handler) createPayment(writer http.ResponseWriter, request *http.Request) {
	var ok bool
	request, ok = withRequestMetadata(writer, request)
	if !ok {
		return
	}
	var input createPaymentRequest
	if !decodeJSON(writer, request, &input) {
		return
	}

	payment, err := h.create.Execute(request.Context(), paymentapplication.CreatePaymentCommand{
		ID:       paymentdomain.ID(input.ID),
		Amount:   input.Amount,
		Currency: paymentdomain.Currency(input.Currency),
	})
	if err != nil {
		writePaymentError(writer, err)
		return
	}

	writer.Header().Set("Location", "/payments/"+url.PathEscape(string(payment.ID())))
	api.WriteJSON(writer, http.StatusCreated, newPaymentResponse(payment))
}

func (h *Handler) getPayment(writer http.ResponseWriter, request *http.Request) {
	var ok bool
	request, ok = withRequestMetadata(writer, request)
	if !ok {
		return
	}
	id, ok := paymentID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "payment route not found")
		return
	}

	payment, err := h.get.Execute(request.Context(), paymentdomain.ID(id))
	if err != nil {
		writePaymentError(writer, err)
		return
	}

	api.WriteJSON(writer, http.StatusOK, newPaymentResponse(payment))
}

func (h *Handler) command(writer http.ResponseWriter, request *http.Request) {
	var ok bool
	request, ok = withRequestMetadata(writer, request)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(request.URL.Path, paymentPathPrefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		api.WriteError(writer, http.StatusNotFound, "not_found", "payment route not found")
		return
	}

	id, err := url.PathUnescape(parts[0])
	if err != nil || id == "" {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "invalid payment id")
		return
	}

	var payment *paymentdomain.Payment
	switch parts[1] {
	case "authorize":
		payment, err = h.authorize.Execute(request.Context(), paymentapplication.AuthorizePaymentCommand{PaymentID: paymentdomain.ID(id)})
	case "capture":
		var input amountRequest
		if !decodeJSON(writer, request, &input) {
			return
		}
		payment, err = h.capture.Execute(request.Context(), paymentapplication.CapturePaymentCommand{
			PaymentID: paymentdomain.ID(id), Amount: input.Amount, Currency: paymentdomain.Currency(input.Currency),
		})
	case "refund":
		var input amountRequest
		if !decodeJSON(writer, request, &input) {
			return
		}
		payment, err = h.refund.Execute(request.Context(), paymentapplication.RefundPaymentCommand{
			PaymentID: paymentdomain.ID(id), Amount: input.Amount, Currency: paymentdomain.Currency(input.Currency),
		})
	case "cancel":
		payment, err = h.cancel.Execute(request.Context(), paymentapplication.CancelPaymentCommand{PaymentID: paymentdomain.ID(id)})
	default:
		api.WriteError(writer, http.StatusNotFound, "not_found", "payment route not found")
		return
	}
	if err != nil {
		writePaymentError(writer, err)
		return
	}

	api.WriteJSON(writer, http.StatusOK, newPaymentResponse(payment))
}

func withRequestMetadata(writer http.ResponseWriter, request *http.Request) (*http.Request, bool) {
	correlationID := strings.TrimSpace(request.Header.Get("X-Correlation-ID"))
	if correlationID == "" {
		var err error
		correlationID, err = observability.NewCorrelationID()
		if err != nil {
			api.WriteError(writer, http.StatusInternalServerError, "internal_error", err.Error())
			return request, false
		}
	}
	if len(correlationID) > 128 || strings.IndexFunc(correlationID, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "X-Correlation-ID must be a printable value of at most 128 characters")
		return request, false
	}
	writer.Header().Set("X-Correlation-ID", correlationID)
	return request.WithContext(observability.WithMetadata(request.Context(), observability.Metadata{CorrelationID: correlationID})), true
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) bool {
	if !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
		api.WriteError(writer, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}

	if request.Body == nil {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "request body is required")
		return false
	}

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
		return false
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "request body must contain one JSON object")
		return false
	}

	return true
}

func paymentID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, paymentPathPrefix), "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}

	id, err := url.PathUnescape(parts[0])
	return id, err == nil && id != ""
}

func newPaymentResponse(payment *paymentdomain.Payment) paymentResponse {
	return paymentResponse{
		ID:               string(payment.ID()),
		Amount:           payment.Amount().Amount(),
		Currency:         string(payment.Amount().Currency()),
		Status:           string(payment.Status()),
		AuthorizedAmount: payment.AuthorizedAmount().Amount(),
		CapturedAmount:   payment.CapturedAmount().Amount(),
		RefundedAmount:   payment.RefundedAmount().Amount(),
		Version:          payment.Version(),
	}
}

func writePaymentError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	switch {
	case errors.Is(err, paymentapplication.ErrPaymentNotFound):
		status = http.StatusNotFound
		code = "payment_not_found"
	case errors.Is(err, paymentdomain.ErrInvalidTransition),
		errors.Is(err, paymentdomain.ErrInvalidCapturedAmount),
		errors.Is(err, paymentdomain.ErrInvalidRefundedAmount),
		errors.Is(err, paymentapplication.ErrPaymentVersionConflict):
		status = http.StatusConflict
		code = "payment_operation_rejected"
	case errors.Is(err, paymentdomain.ErrInvalidPaymentID),
		errors.Is(err, paymentdomain.ErrInvalidMoneyAmount),
		errors.Is(err, paymentdomain.ErrInvalidCurrency):
		status = http.StatusBadRequest
		code = "invalid_request"
	}

	api.WriteError(writer, status, code, err.Error())
}
