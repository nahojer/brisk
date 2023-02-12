package brisk_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nahojer/brisk"
)

var tests = []struct {
	RouteMethod  string
	RoutePattern string

	Method string
	Path   string
	Match  bool
	Params map[string]string
}{
	// prefix
	{
		"GET", "/not-prefix",
		"GET", "/not-prefix/anything/else", false, nil,
	},
	{
		"GET", "/prefixdots...",
		"GET", "/prefixdots/anything/else", true, nil,
	},
	{
		"GET", "/prefixdots/...",
		"GET", "/prefixdots", true, nil,
	},
	// path params
	{
		"GET", "/path-param/:id",
		"GET", "/path-param/123", true, map[string]string{"id": "123"},
	},
	{
		"GET", "/path-params/:era/:group/:member",
		"GET", "/path-params/60s/beatles/lennon", true, map[string]string{
			"era":    "60s",
			"group":  "beatles",
			"member": "lennon",
		},
	},
}

func TestRouter(t *testing.T) {
	for _, tt := range tests {
		r := brisk.NewRouter()

		var (
			match    bool
			matchReq *http.Request
		)
		r.Handle(tt.RouteMethod, tt.RoutePattern, func(w http.ResponseWriter, r *http.Request) error {
			match = true
			matchReq = r
			return nil
		})

		req := httptest.NewRequest(tt.Method, tt.Path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if match != tt.Match {
			t.Errorf("%q %q: got match %t, want %t", tt.Method, tt.Path, tt.Match, match)
		}

		for paramName, wantParamVal := range tt.Params {
			gotParamVal := brisk.Param(matchReq, paramName)
			if gotParamVal != wantParamVal {
				t.Errorf("httprouter.Param(matchReq, %q) = %q, want %q", paramName, gotParamVal, wantParamVal)
			}
		}
	}
}

func TestRouter_Middleware(t *testing.T) {
	var got string
	mw := func(text string) brisk.Middleware {
		return func(next brisk.Handler) brisk.Handler {
			return func(w http.ResponseWriter, r *http.Request) error {
				got += text
				return next(w, r)
			}
		}
	}
	h := func(w http.ResponseWriter, r *http.Request) error {
		got += "h"
		return nil
	}

	r := brisk.NewRouter(mw("1"), mw("2"))
	r.Get("/", h, mw("3"), mw("4"))
	mygroup := r.Group("mygroup", mw("5"), mw("6"))
	mygroup.Get("/", h, mw("7"), mw("8"))

	for _, path := range []string{"/", "/mygroup/"} {
		req := httptest.NewRequest("GET", "http://localhost"+path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	want := "1234h125678h"
	if got != want {
		t.Errorf("Got %q, want %q", got, want)
	}
}

func TestRouter_NotFoundHandler(t *testing.T) {
	r := brisk.NewRouter()
	r.NotFoundHandler = func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "not found")
		return nil
	}

	// We havn't registered a route handler yet, so all requests should return 404.
	req := httptest.NewRequest("GET", "http://localhost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	wantCode := http.StatusNotFound
	if gotCode := w.Code; gotCode != wantCode {
		t.Errorf("Got status code %d, want %d", w.Code, http.StatusNotFound)
	}

	wantBody := "not found"
	if gotBody := w.Body.String(); gotBody != wantBody {
		t.Errorf("Got body %q, want %q", w.Code, http.StatusNotFound)
	}
}

func TestRouter_ErrorHandler(t *testing.T) {
	r := brisk.NewRouter()
	r.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}

	r.Get("/...", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("some error")
	})

	req := httptest.NewRequest("GET", "http://localhost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	wantCode := http.StatusInternalServerError
	if gotCode := w.Code; gotCode != wantCode {
		t.Errorf("Got status code %d, want %d", w.Code, http.StatusNotFound)
	}

	wantBody := "some error"
	if gotBody := w.Body.String(); gotBody != wantBody {
		t.Errorf("Got body %q, want %q", w.Code, http.StatusNotFound)
	}
}
