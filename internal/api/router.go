package api

import (
	"encoding/json"
	"net/http"
	"sort"
)

type Router struct {
	routes map[string]map[string]http.Handler
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]map[string]http.Handler)}
}

func (r *Router) Handle(method, path string, handler http.Handler) error {
	if method == "" {
		return ErrInvalidMethod
	}
	if path == "" || path[0] != '/' {
		return ErrInvalidPath
	}
	if handler == nil {
		return ErrNilHandler
	}

	if r.routes[path] == nil {
		r.routes[path] = make(map[string]http.Handler)
	}
	if _, exists := r.routes[path][method]; exists {
		return ErrDuplicateRoute
	}

	r.routes[path][method] = handler
	return nil
}

func (r *Router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	methods, exists := r.routes[request.URL.Path]
	if !exists {
		writeError(writer, http.StatusNotFound, "not_found", "route not found")
		return
	}

	handler, exists := methods[request.Method]
	if !exists {
		allowed := make([]string, 0, len(methods))
		for method := range methods {
			allowed = append(allowed, method)
		}
		sort.Strings(allowed)
		writer.Header().Set("Allow", joinMethods(allowed))
		writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	handler.ServeHTTP(writer, request)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, ErrorResponse{
		Error: APIError{Code: code, Message: message},
	})
}

func joinMethods(methods []string) string {
	result := ""
	for index, method := range methods {
		if index > 0 {
			result += ", "
		}
		result += method
	}

	return result
}
