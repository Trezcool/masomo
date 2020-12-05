package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/trezcool/masomo/backend/apps/api/echo"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
)

var (
	// todo: load from test config
	appName                   = "Masomo"
	secretKey                 = []byte("secret")
	serverName                = "localhost"
	defaultFromEmail          = "noreply@" + serverName
	jwtExpirationDelta        = 10 * time.Minute
	jwtRefreshExpirationDelta = 4 * time.Hour
	passwordResetTimeoutDelta = 3 * 24 * time.Hour

	usrRepo user.Repository

	errMissingToken = httpErr{Error: "missing or malformed jwt"}
)

func setup(t *testing.T) Server {
	// set up DB
	db, err := dummydb.Open()
	if err != nil {
		t.Fatalf("setup() failed: %v", err)
	}
	usrRepo = dummydb.NewUserRepository(db)

	// set up services
	mailSvc := dummymail.NewServiceMock(appName, defaultFromEmail)
	usrSvc := user.NewServiceMock(usrRepo, mailSvc, secretKey, passwordResetTimeoutDelta)

	// set up server
	app := NewServer(
		&Options{
			Debug:                     false,
			DisableReqLogs:            true,
			AppName:                   appName,
			SecretKey:                 secretKey,
			JwtExpirationDelta:        jwtExpirationDelta,
			JwtRefreshExpirationDelta: jwtRefreshExpirationDelta,
			UserSvc:                   usrSvc,
		},
	)
	return app
}

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

func getToken(t *testing.T, usr user.User) string {
	claims := GetUserClaims(usr)
	token, err := GenerateToken(claims)
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
