package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"

	"github.com/trezcool/masomo/backend/apps/api/echo/helpers"
	"github.com/trezcool/masomo/backend/core/user"
)

var (
	// todo: load from config
	appName                   = "Masomo"
	secretKey                 = []byte("secret")
	serverName                = "localhost"
	defaultFromEmail          = "noreply@" + serverName
	jwtExpirationDelta        = 10 * time.Minute
	jwtRefreshExpirationDelta = 4 * time.Hour
	passwordResetTimeoutDelta = 3 * 24 * time.Hour

	errMissingToken = httpErr{Error: "missing or malformed jwt"}
)

type httpErr struct {
	Error string `json:"error"`
}

type httpTest struct {
	name     string
	method   string
	path     string
	body     []byte
	token    string
	wantCode int
	wantData []byte
	extra    interface{}
}

func newAuthRequest(method, path, token string, data ...[]byte) (*http.Request, *httptest.ResponseRecorder) {
	var body bytes.Buffer
	if len(data) > 0 {
		body.Write(data[0])
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	return req, rec
}

func newRequest(method, path string, data ...[]byte) (*http.Request, *httptest.ResponseRecorder) {
	return newAuthRequest(method, path, "", data...)
}

func initApp() (*echo.Echo, *echo.Group, echo.MiddlewareFunc) {
	app := echo.New()
	app.Pre(middleware.RemoveTrailingSlash())
	app.HTTPErrorHandler = helpers.AppHTTPErrorHandler
	v1 := app.Group("/v1")
	jwt := helpers.ConfigureAuth(appName, secretKey, jwtExpirationDelta, jwtRefreshExpirationDelta)
	return app, v1, jwt
}

func getToken(t *testing.T, usr user.User) string {
	claims := helpers.GetUserClaims(usr)
	token, err := helpers.GenerateToken(claims)
	if err != nil {
		t.Fatalf("getToken() failed: %v", err)
	}
	return token
}

func marchallObj(t *testing.T, obj interface{}) []byte {
	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marchallList() failed: %v", err)
	}
	return data
}

func marchallList(t *testing.T, objs ...interface{}) []byte {
	data, err := json.Marshal(objs)
	if err != nil {
		t.Fatalf("marchallList() failed: %v", err)
	}
	return data
}

func jsonBytesEqual(t *testing.T, b1, b2 []byte) (bool, error) {
	var j1, j2 interface{}
	if err := json.Unmarshal(b1, &j1); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b2, &j2); err != nil {
		return false, err
	}
	if reflect.DeepEqual(j1, j2) {
		return true, nil
	}
	return assert.ElementsMatch(t, j1, j2), nil
}

func checkCodeAndData(t *testing.T, tt httpTest, rec *httptest.ResponseRecorder) {
	if rec.Code != tt.wantCode {
		t.Errorf("failed! code = %v; wantCode %v", rec.Code, tt.wantCode)
	}
	ok, err := jsonBytesEqual(t, rec.Body.Bytes(), tt.wantData)
	if err != nil {
		t.Errorf("jsonBytesEqual() failed to compare; err %v", err)
	}
	if !ok {
		t.Errorf("failed! data = %v; wantData %v", rec.Body.String(), string(tt.wantData))
	}
}
