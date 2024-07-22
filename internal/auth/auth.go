package auth

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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

const UsersFile = "../../data/users.gob"

func init() {
	err := loadUsers()
	if err != nil {
		log.Printf("Failed to load users: %v", err)
	}
}

func loadUsers() error {
	file, err := os.Open(UsersFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Users file does not exist. Starting with an empty user list.")
			return nil // Not an error, just a new application
		}
		return fmt.Errorf("error opening users file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&Users); err != nil {
		return fmt.Errorf("error decoding users: %w", err)
	}

	log.Printf("Loaded %d users from file", len(Users))
	return nil
}

func saveUsers() error {
	file, err := os.Create(UsersFile)
	if err != nil {
		return fmt.Errorf("error creating users file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(Users); err != nil {
		return fmt.Errorf("error encoding users: %w", err)
	}

	log.Printf("Saved %d users to file", len(Users))
	return nil
}

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
			Secure:   true,
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
			log.Printf("Error hashing password: %v", err)
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

		if err := saveUsers(); err != nil {
			log.Printf("Failed to save users: %v", err)
			http.Error(w, "Failed to save user data", http.StatusInternalServerError)
			return
		}

		log.Printf("New user registered: %s", username)
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