package auth

import (
	"fmt"
	"net/http"

	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

type Auth struct {
	cache redis.Conn
	users map[string]string
}

func InitAuth() (*Auth, error) {
	conn, err := redis.DialURL("redis://localhost")
	if err != nil {
		return nil, err
	}

	a := Auth{
		cache: conn,
		users: map[string]string{
			"ledu": "password1",
			"mati": "password2",
		},
	}

	return &a, nil
}

/*
	Takes username and password as input and returns
	session token if credentials are valid and appropriate http status code
*/
func (a *Auth) Signin(username string, password string) (string, int) {
	expectedPassword, ok := a.users[username]

	if !ok || expectedPassword != password {
		return "", http.StatusUnauthorized
	}

	sessionToken := uuid.NewV4().String()

	_, err := a.cache.Do("SETEX", sessionToken, "120", username)
	if err != nil {
		return "", http.StatusInternalServerError
	}

	return sessionToken, http.StatusOK
}

func (a *Auth) Authorize(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return ""
		}
		w.WriteHeader(http.StatusBadRequest)
		return ""
	}
	token := c.Value

	response, err := a.cache.Do("GET", token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return ""
	}
	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return ""
	}

	return fmt.Sprintf("%s", response)
}