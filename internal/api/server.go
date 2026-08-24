package api

import (
	"context"
	"net/http"
	"sync/atomic"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type Server struct {
	httpServer *http.Server
	router     *Router
	ready      atomic.Bool
}

func NewServer(address string) (*Server, error) {
	if address == "" {
		return nil, ErrInvalidAddress
	}

	server := &Server{router: NewRouter()}
	server.ready.Store(true)

	if err := server.router.Handle(http.MethodGet, "/health/live", http.HandlerFunc(server.live)); err != nil {
		return nil, err
	}
	if err := server.router.Handle(http.MethodGet, "/health/ready", http.HandlerFunc(server.readyCheck)); err != nil {
		return nil, err
	}

	server.httpServer = &http.Server{
		Addr:    address,
		Handler: server.router,
	}

	return server, nil
}

func (s *Server) Handle(method, path string, handler http.Handler) error {
	return s.router.Handle(method, path, handler)
}

func (s *Server) HandlePrefix(method, prefix string, handler http.Handler) error {
	return s.router.HandlePrefix(method, prefix, handler)
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.router.ServeHTTP(writer, request)
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

func (s *Server) live(writer http.ResponseWriter, _ *http.Request) {
	WriteJSON(writer, http.StatusOK, HealthResponse{Status: "alive"})
}

func (s *Server) readyCheck(writer http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		WriteJSON(writer, http.StatusServiceUnavailable, HealthResponse{Status: "not_ready"})
		return
	}

	WriteJSON(writer, http.StatusOK, HealthResponse{Status: "ready"})
}
