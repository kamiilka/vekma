package httpapi

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	userKey contextKey = "user"
	roleKey contextKey = "role"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if raw == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		token := strings.TrimPrefix(raw, "Bearer ")
		if token == raw {
			writeError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		claims, err := s.tokens.ParseToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userKey, claims.Subject)
		ctx = context.WithValue(ctx, roleKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requirePermission(permission string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := roleFromContext(r.Context())
			if !s.store.HasPermission(role, permission) {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) hasWarehouseAccess(r *http.Request, warehouseID int64) bool {
	return s.store.HasWarehouseAccess(userFromContext(r.Context()), roleFromContext(r.Context()), warehouseID)
}

func (s *Server) hasCashboxAccess(r *http.Request, cashboxID int64) bool {
	return s.store.HasCashboxAccess(userFromContext(r.Context()), roleFromContext(r.Context()), cashboxID)
}

func userFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userKey).(string)
	return value
}

func roleFromContext(ctx context.Context) string {
	value, _ := ctx.Value(roleKey).(string)
	return value
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
