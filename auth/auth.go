package auth

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	l "github.com/noisersup/encryptedfs-api/logger"
	"github.com/noisersup/encryptedfs-api/models"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	cache *redis.Pool
	db    models.Database
}

func InitAuth(userDb models.Database) (*Auth, error) {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL("redis://localhost")
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	a := Auth{
		cache: pool,
		db:    userDb,
	}

	return &a, nil
}

/*
	Takes username and password as input and returns
	session token if credentials are valid and appropriate http status code
*/
func (a *Auth) Signin(username string, password string) (string, int) {
	var expectedPassword string
	expectedPassword, err := a.db.GetPasswordOfUser(username)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return "", http.StatusUnauthorized
		}
		l.Err("db.Query error: %s", err.Error())
		return "", http.StatusInternalServerError
	}

	if err = bcrypt.CompareHashAndPassword([]byte(expectedPassword), []byte(password)); err != nil {
		return "", http.StatusUnauthorized
	}

	sessionToken := uuid.NewV4().String()

	conn := a.cache.Get()
	_, err = conn.Do("SETEX", sessionToken, "120", username)
	conn.Close()
	if err != nil {
		l.Err("redis error: %s", err.Error())
		return "", http.StatusInternalServerError
	}

	return sessionToken, http.StatusOK
}

func (a *Auth) Signup(username string, password string) int {
	/*
		verify password and username len
	*/
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		log.Print("bcrypt: ", err)
		return http.StatusInternalServerError
	}

	err = a.db.NewUser(username, string(hash))

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return http.StatusConflict
		} else {
			log.Print("db: ", err)
			return http.StatusInternalServerError
		}
	}

	return http.StatusOK
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

	conn := a.cache.Get()
	response, err := conn.Do("GET", token)
	conn.Close()
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

func (a *Auth) Refresh(r *http.Request) (*http.Cookie, int) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, http.StatusUnauthorized
		}
		return nil, http.StatusBadRequest
	}

	userToken := cookie.Value

	conn := a.cache.Get()
	response, err := conn.Do("GET", userToken)
	if err != nil {
		conn.Close()
		return nil, http.StatusInternalServerError
	}
	if response == nil {
		conn.Close()
		return nil, http.StatusUnauthorized
	}

	newToken := uuid.NewV4().String()
	_, err = conn.Do("SETEX", newToken, "120", fmt.Sprintf("%s", response))
	if err != nil {
		conn.Close()
		return nil, http.StatusInternalServerError
	}
	_, err = conn.Do("DEL", userToken)
	conn.Close()
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	return &http.Cookie{
		Name:    "session_token",
		Value:   newToken,
		Expires: time.Now().Add(120 * time.Second),
	}, http.StatusOK
}
