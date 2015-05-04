package restapi

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/ant0ine/go-json-rest/rest"
)

// AuthBasicMiddleware provides a simple AuthBasic implementation. On failure, a 401 HTTP response
//is returned. On success, the wrapped middleware is called, and the userId is made available as
// request.Env["REMOTE_USER"].(string)
type AuthBasicMiddleware struct {

	// Realm name to display to the user. Required.
	Realm string

	// Callback function that should perform the authentication of the user based on userId and
	// password. Must return true on success, false on failure. Required.
	Authenticator func(userId string, password string, request *rest.Request) bool

	// Callback function that should perform the authorization of the authenticated user. Called
	// only after an authentication success. Must return true on success, false on failure.
	// Optional, default to success.
	Authorizator func(userId string, request *rest.Request) bool
}

// MiddlewareFunc makes AuthBasicMiddleware implement the Middleware interface.
func (mw *AuthBasicMiddleware) MiddlewareFunc(handler rest.HandlerFunc) rest.HandlerFunc {

	if mw.Realm == "" {
		log.Fatal("Realm is required")
	}

	if mw.Authenticator == nil {
		log.Fatal("Authenticator is required")
	}

	if mw.Authorizator == nil {
		mw.Authorizator = func(userId string, request *rest.Request) bool {
			return true
		}
	}

	return func(writer rest.ResponseWriter, request *rest.Request) {

		authHeader := request.Header.Get("Authorization")
		if authHeader == "" {
			mw.unauthorized(writer)
			return
		}

		providedUserID, providedPassword, err := mw.decodeBasicAuthHeader(authHeader)

		if err != nil {
			rest.Error(writer, "Invalid authentication", http.StatusBadRequest)
			return
		}

		if !mw.Authenticator(providedUserID, providedPassword, request) {
			mw.unauthorized(writer)
			return
		}

		if !mw.Authorizator(providedUserID, request) {
			mw.unauthorized(writer)
			return
		}

		request.Env["REMOTE_USER"] = providedUserID

		handler(writer, request)
	}
}

func (mw *AuthBasicMiddleware) unauthorized(writer rest.ResponseWriter) {
	writer.Header().Set("WWW-Authenticate", "Basic realm="+mw.Realm)
	rest.Error(writer, "Not Authorized", http.StatusUnauthorized)
}

func (mw *AuthBasicMiddleware) decodeBasicAuthHeader(header string) (user string, password string, err error) {

	parts := strings.SplitN(header, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Basic") {
		return "", "", errors.New("Invalid authentication")
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", errors.New("Invalid base64")
	}

	creds := strings.SplitN(string(decoded), ":", 2)
	if len(creds) != 2 {
		return "", "", errors.New("Invalid authentication")
	}

	return creds[0], creds[1], nil
}
