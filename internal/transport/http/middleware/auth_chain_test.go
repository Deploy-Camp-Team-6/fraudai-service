package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	var result string
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result += "A"
			next.ServeHTTP(w, r)
			result += "C"
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result += "B"
			next.ServeHTTP(w, r)
		})
	}
	chained := Chain(mw1, mw2)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	chained(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	assert.Equal(t, "ABC", result, "middleware should be chained in the correct order")
}

func TestFinalAuthCheck(t *testing.T) {
	t.Run("with identity", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)

		ctx := req.Context()
		ctx = context.WithValue(ctx, ctxKeyIdentity, Identity{UserID: 1})
		req = req.WithContext(ctx)

		finalAuthCheck(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("without identity", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)

		finalAuthCheck(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
