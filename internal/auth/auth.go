package auth

import (
	"html/template"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var Templates = template.Must(template.ParseGlob("../../internal/public/*.html"))

type User struct {
	Username string
	Password []byte
	Email    string
}

var (
	Users      = make(map[string]User)
	UsersMutex sync.RWMutex
)

func SigninHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		UsersMutex.RLock()
		user, exists := Users[username]
		UsersMutex.RUnlock()

		if !exists || bcrypt.CompareHashAndPassword(user.Password, []byte(password)) != nil {
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "Invalid username or password",
			}
			Templates.ExecuteTemplate(w, "signin.html", data)
			return
		}

		sessionToken := generateSessionToken()

		SessionsMutex.Lock()
		Sessions[sessionToken] = Session{
			Username: username,
			Expiry:   time.Now().Add(24 * time.Hour),
		}
		SessionsMutex.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   true, // Requires HTTPS
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	Templates.ExecuteTemplate(w, "signin.html", nil)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("email")

		UsersMutex.RLock()
		_, exists := Users[username]
		UsersMutex.RUnlock()

		if exists {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		user := User{
			Username: username,
			Password: hashedPassword,
			Email:    email,
		}

		UsersMutex.Lock()
		Users[username] = user
		UsersMutex.Unlock()

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	Templates.ExecuteTemplate(w, "register.html", nil)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	SessionsMutex.Lock()
	delete(Sessions, session.Value)
	SessionsMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
