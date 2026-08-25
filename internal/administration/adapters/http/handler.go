package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"proxynth/payment-sandbox/internal/api"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

type clockPort interface {
	Advance(time.Duration) error
	Now() time.Time
}

type providerRegistry interface {
	IDs() []providerdomain.ProviderID
}

type Handler struct {
	clock    clock.Clock
	advancer *clock.TimeAdvancer
	registry providerRegistry
}

func NewHandler(clockSource clockPort, registry providerRegistry) (*Handler, error) {
	if clockSource == nil {
		return nil, ErrNilClock
	}
	if registry == nil {
		return nil, ErrNilRegistry
	}

	advancer, err := clock.NewTimeAdvancer(clockSource)
	if err != nil {
		return nil, err
	}

	return &Handler{
		clock:    clockSource,
		advancer: advancer,
		registry: registry,
	}, nil
}

func (h *Handler) Register(server *api.Server, token string) error {
	if err := server.HandleAdmin(http.MethodGet, "/admin/time", http.HandlerFunc(h.getTime), token); err != nil {
		return err
	}
	if err := server.HandleAdmin(http.MethodPost, "/admin/time/advance", http.HandlerFunc(h.advanceTime), token); err != nil {
		return err
	}
	return server.HandleAdmin(http.MethodGet, "/admin/providers", http.HandlerFunc(h.listProviders), token)
}

type timeResponse struct {
	Current time.Time `json:"current"`
}

type advanceTimeRequest struct {
	By string `json:"by"`
}

type providersResponse struct {
	Providers []string `json:"providers"`
}

func (h *Handler) getTime(writer http.ResponseWriter, _ *http.Request) {
	api.WriteJSON(writer, http.StatusOK, timeResponse{Current: h.clock.Now().UTC()})
}

func (h *Handler) advanceTime(writer http.ResponseWriter, request *http.Request) {
	var input advanceTimeRequest
	if !decodeJSON(writer, request, &input) {
		return
	}

	duration, err := time.ParseDuration(input.By)
	if err != nil || duration <= 0 {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "by must be a positive duration")
		return
	}

	result, err := h.advancer.Execute(clock.AdvanceTimeCommand{By: duration})
	if err != nil {
		writeAdministrationError(writer, err)
		return
	}

	api.WriteJSON(writer, http.StatusOK, timeResponse{Current: result.Current.UTC()})
}

func (h *Handler) listProviders(writer http.ResponseWriter, _ *http.Request) {
	ids := h.registry.IDs()
	providers := make([]string, 0, len(ids))
	for _, id := range ids {
		providers = append(providers, string(id))
	}

	api.WriteJSON(writer, http.StatusOK, providersResponse{Providers: providers})
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

func writeAdministrationError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	if errors.Is(err, clock.ErrInvalidAdvance) || errors.Is(err, clock.ErrBackwardAdvance) {
		status = http.StatusConflict
		code = "time_advance_rejected"
	}

	api.WriteError(writer, status, code, err.Error())
}
