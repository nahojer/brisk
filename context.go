package brisk

import "net/http"

type ctxKey string

// Param returns the path parameter for the request with given name. Returns
// the empty string if name not found.
func Param(r *http.Request, name string) string {
	v, ok := r.Context().Value(ctxKey(name)).(string)
	if !ok {
		return ""
	}
	return v
}
