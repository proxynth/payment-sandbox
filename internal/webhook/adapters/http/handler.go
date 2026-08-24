package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"proxynth/payment-sandbox/internal/api"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

const endpointPathPrefix = "/webhook-endpoints/"

type Handler struct {
	register *webhookapplication.RegisterEndpoint
	get      *webhookapplication.GetEndpoint
	list     *webhookapplication.ListEndpoints
}

func NewHandler(repository webhookapplication.Repository) (*Handler, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &Handler{
		register: webhookapplication.NewRegisterEndpoint(repository),
		get:      webhookapplication.NewGetEndpoint(repository),
		list:     webhookapplication.NewListEndpoints(repository),
	}, nil
}

func (h *Handler) Register(server *api.Server) error {
	if err := server.Handle(http.MethodPost, "/webhook-endpoints", http.HandlerFunc(h.registerEndpoint)); err != nil {
		return err
	}
	if err := server.Handle(http.MethodGet, "/webhook-endpoints", http.HandlerFunc(h.listEndpoints)); err != nil {
		return err
	}
	return server.HandlePrefix(http.MethodGet, endpointPathPrefix, http.HandlerFunc(h.getEndpoint))
}

type endpointRequest struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type endpointResponse struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

func (h *Handler) registerEndpoint(writer http.ResponseWriter, request *http.Request) {
	var input endpointRequest
	if !decodeJSON(writer, request, &input) {
		return
	}

	endpoint, err := h.register.Execute(request.Context(), webhookapplication.RegisterEndpointCommand{
		ID: webhookdomain.EndpointID(input.ID), URL: input.URL,
	})
	if err != nil {
		writeEndpointError(writer, err)
		return
	}

	writer.Header().Set("Location", "/webhook-endpoints/"+url.PathEscape(string(endpoint.ID())))
	api.WriteJSON(writer, http.StatusCreated, newEndpointResponse(endpoint))
}

func (h *Handler) getEndpoint(writer http.ResponseWriter, request *http.Request) {
	id, ok := endpointID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "webhook endpoint route not found")
		return
	}

	endpoint, err := h.get.Execute(request.Context(), webhookdomain.EndpointID(id))
	if err != nil {
		writeEndpointError(writer, err)
		return
	}

	api.WriteJSON(writer, http.StatusOK, newEndpointResponse(endpoint))
}

func (h *Handler) listEndpoints(writer http.ResponseWriter, request *http.Request) {
	endpoints, err := h.list.Execute(request.Context())
	if err != nil {
		writeEndpointError(writer, err)
		return
	}

	response := make([]endpointResponse, 0, len(endpoints))
	for _, endpoint := range endpoints {
		response = append(response, newEndpointResponse(endpoint))
	}

	api.WriteJSON(writer, http.StatusOK, response)
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

func endpointID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, endpointPathPrefix), "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}

	id, err := url.PathUnescape(parts[0])
	return id, err == nil && id != ""
}

func newEndpointResponse(endpoint *webhookdomain.Endpoint) endpointResponse {
	return endpointResponse{ID: string(endpoint.ID()), URL: endpoint.URL(), Enabled: endpoint.Enabled()}
}

func writeEndpointError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	switch {
	case errors.Is(err, webhookapplication.ErrEndpointNotFound):
		status = http.StatusNotFound
		code = "webhook_endpoint_not_found"
	case errors.Is(err, webhookapplication.ErrEndpointAlreadyExists):
		status = http.StatusConflict
		code = "webhook_endpoint_already_exists"
	case errors.Is(err, webhookdomain.ErrInvalidEndpointID),
		errors.Is(err, webhookdomain.ErrInvalidEndpointURL):
		status = http.StatusBadRequest
		code = "invalid_request"
	}

	api.WriteError(writer, status, code, err.Error())
}
