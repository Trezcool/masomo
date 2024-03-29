package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	. "github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/services/email"
	logsvc "github.com/trezcool/masomo/services/logger"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
	"github.com/trezcool/masomo/tests"
)

var (
	db      *sql.DB
	conf    *core.Config
	server  *Server
	usrRepo user.Repository

	errMissingToken = httpErr{Error: "missing or malformed jwt"}
)

func TestMain(m *testing.M) {
	var err error

	// =========================================================================
	// Dependencies
	conf = core.NewConfig()

	logger := logsvc.NewRollbarLogger(log.Default(), conf)
	logger.Enable(false)

	// set up DB & repos
	db = testutil.OpenDB(conf)
	usrRepo = boiledrepos.NewUserRepository(db)

	// set up services
	mailSvc := emailsvc.NewConsoleServiceMock(conf)
	usrSvc := user.NewServiceMock(db, usrRepo, mailSvc, conf)

	// =========================================================================
	// Initialization
	validate := validator.New()
	_en := en.New()
	uni := ut.New(_en, _en)
	translator, _ := uni.GetTranslator("en")
	core.InitValidators(validate, translator)
	user.InitValidators(validate, translator)

	core.ParseEmailTemplates(logger)
	user.LoadCommonPasswords(logger)

	// set up server
	server = NewServer(
		ServerDeps{
			Conf:       conf,
			Logger:     logger,
			UserSvc:    usrSvc,
			Validate:   validate,
			Translator: translator,
		},
	)

	// run tests
	code := m.Run()

	// clean up
	if err = db.Close(); err != nil {
		fmt.Printf("db.Close(): %v", err)
		os.Exit(1)
	}

	os.Exit(code)
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
		t.Fatalf("getToken(): %v", err)
	}
	return token
}

func marchallObj(t *testing.T, obj interface{}) []byte {
	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marchallList(): %v", err)
	}
	return data
}

func marchallList(t *testing.T, objs ...interface{}) []byte {
	data, err := json.Marshal(objs)
	if err != nil {
		t.Fatalf("marchallList(): %v", err)
	}
	return data
}

func jsonBytesEqual(b1, b2 []byte) (bool, error) {
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
	return false, nil
}

func checkCodeAndData(t *testing.T, tt httpTest, rec *httptest.ResponseRecorder) {
	if rec.Code != tt.wantCode {
		t.Errorf("failed! code = %v; wantCode %v", rec.Code, tt.wantCode)
	}
	ok, err := jsonBytesEqual(rec.Body.Bytes(), tt.wantData)
	if err != nil {
		t.Errorf("jsonBytesEqual() failed to compare; err %v", err)
	}
	if !ok {
		t.Errorf("failed! data = %v; wantData %v", rec.Body.String(), string(tt.wantData))
	}
}
