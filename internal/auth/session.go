package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

var (
	Sessions      = make(map[string]Session)
	SessionsMutex = sync.RWMutex{}
)

type Session struct {
	Username string
	Expiry   time.Time
}

func generateSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}

		SessionsMutex.RLock()
		userSession, exists := Sessions[session.Value]
		SessionsMutex.RUnlock()

		if !exists || time.Now().After(userSession.Expiry) {
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}

		// Refresh session
		SessionsMutex.Lock()
		Sessions[session.Value] = Session{
			Username: userSession.Username,
			Expiry:   time.Now().Add(24 * time.Hour),
		}
		SessionsMutex.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    session.Value,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})

		next.ServeHTTP(w, r)
	}
}
