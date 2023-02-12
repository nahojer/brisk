package brisk

import (
	"context"
	"net/http"
	"strings"

	"github.com/nahojer/sage"
)

// Handler is a type that handles HTTP requests.
type Handler func(w http.ResponseWriter, r *http.Request) error

// Router routes HTTP requests. Must be initialized by calling [NewRouter].
type Router struct {
	// NotFoundHandler is the handler to call when no routes match. By default
	// uses a handler that writes status code 404 to the HTTP header.
	NotFoundHandler Handler
	// ErrorHandler is the function to call when a Handler returns a non-nil
	// error. By default is nil and does nothing.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
	mw           []Middleware
	prefix       string // Path pattern prefix without leading/trailing slashes ("/")
	routes       *sage.RoutesTrie[Handler]
}

// NewRouter creates a new router with default values. Passed middleware will
// be executed for all handlers registered on the router in the order they
// are provided.
func NewRouter(mw ...Middleware) *Router {
	return &Router{
		NotFoundHandler: func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusNotFound)
			return nil
		},
		mw:     mw,
		routes: sage.NewRoutesTrie[Handler](),
	}
}

// Handle registers the handler to run for a given HTTP method and path
// pair. Middleware are executed in the order they are provided after
// any global middleware has been executed.
//
// Path parameters are specified by prefixing path segments with a colon (":").
// To match any path that has a specific prefix, use the three dots ("...") prefix
// indicator. Examples:
//
//	// Call handleImages for any path prefixed with /images.
//	router.Handle("GET", "/images...", handleImages)
//
//	// Path parameter with name "id".
//	router.Handle("GET", "/users/:id", handleGetUser)
//
// Use [Param] to get the value of the path parameter from the request.
func (r *Router) Handle(method, pattern string, h Handler, mw ...Middleware) {
	h = wrapMiddleware(mw, h)
	h = wrapMiddleware(r.mw, h)
	r.routes.Add(method, "/"+r.prefix+pattern, h)
}

// Post calls Handle("DELETE", pattern, h, mw...).
func (r *Router) Delete(pattern string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodDelete, pattern, h, mw...)
}

// Post calls Handle("GET", pattern, h, mw...).
func (r *Router) Get(pattern string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodGet, pattern, h, mw...)
}

// Post calls Handle("PATCH", pattern, h, mw...).
func (r *Router) Patch(pattern string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodPatch, pattern, h, mw...)
}

// Post calls Handle("POST", pattern, h, mw...).
func (r *Router) Post(pattern string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodPost, pattern, h, mw...)
}

// Post calls Handle("PUT", pattern, h, mw...).
func (r *Router) Put(pattern string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodPut, pattern, h, mw...)
}

// ServeHTTP implements the [http.Handler] interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h, params, found := r.routes.Lookup(req)
	if !found {
		if err := r.NotFoundHandler(w, req); err != nil && r.ErrorHandler != nil {
			r.ErrorHandler(w, req, err)
		}
		return
	}

	for k, v := range params {
		ctx := context.WithValue(req.Context(), ctxKey(k), v)
		req = req.WithContext(ctx)
	}

	if err := h(w, req); err != nil && r.ErrorHandler != nil {
		r.ErrorHandler(w, req, err)
	}
}

// Group creates a sub-router with given name. All handlers registered on this
// router will have their path prefixed with name. Provided middleware will be
// appended to the slice of middlewares inherited by the parent router and
// exectuted in the order they are provided.
func (r *Router) Group(name string, mw ...Middleware) *Router {
	return &Router{
		NotFoundHandler: r.NotFoundHandler,
		ErrorHandler:    r.ErrorHandler,
		mw:              append(append([]Middleware{}, r.mw...), mw...),
		routes:          r.routes,
		prefix:          strings.Trim(r.prefix+"/"+name, "/"),
	}
}
