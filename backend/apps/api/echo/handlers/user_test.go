package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"

	"github.com/trezcool/masomo/backend/apps/api/echo/helpers"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
)

var errMissingToken = httpErr{Error: "missing or malformed jwt"}

type httpErr struct {
	Error string `json:"error"`
}

func initEcho(svc *user.Service) *echo.Echo {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.HTTPErrorHandler = helpers.AppHTTPErrorHandler
	v1 := e.Group("/v1")
	jwtMid := middleware.JWTWithConfig(helpers.AppJWTConfig)
	RegisterUserAPI(v1, jwtMid, svc)
	return e
}

func setup(t *testing.T) (*echo.Echo, user.Repository) {
	db, err := dummydb.Open()
	if err != nil {
		t.Fatalf("setup() failed: %v", err)
	}
	repo := dummydb.NewUserRepository(db)
	svc := user.NewService(repo)
	e := initEcho(svc)
	return e, repo
}

func newAuthRequest(method, path, token string, data ...[]byte) (*http.Request, *httptest.ResponseRecorder) {
	var body bytes.Buffer
	if len(data) > 0 {
		body.Write(data[0])
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	return req, rec
}

func newRequest(method, path string, data ...[]byte) (*http.Request, *httptest.ResponseRecorder) {
	return newAuthRequest(method, path, "", data...)
}

func createUser(
	t *testing.T,
	repo user.Repository,
	name, uname, email, pwd string,
	roles []string,
	isActive bool,
	createdAt ...time.Time,
) user.User {
	tstamp := time.Now().UTC()
	if len(createdAt) > 0 {
		tstamp = createdAt[0].UTC()
	}
	usr := user.User{
		Name:      name,
		Username:  uname,
		Email:     email,
		Roles:     roles,
		IsActive:  isActive,
		CreatedAt: tstamp,
		UpdatedAt: tstamp,
	}
	if pwd != "" {
		if err := usr.SetPassword(pwd); err != nil {
			t.Fatalf("createUser() failed: %v", err)
		}
	}
	usr, err := repo.CreateUser(usr)
	if err != nil {
		t.Fatalf("createUser() failed: %v", err)
	}
	return usr
}

func getToken(t *testing.T, usr user.User) string {
	claims := helpers.GetUserClaims(usr)
	token, err := helpers.GenerateToken(claims)
	if err != nil {
		t.Fatalf("getToken() failed: %v", err)
	}
	return token
}

func marchallList(t *testing.T, objs ...interface{}) []byte {
	data, err := json.Marshal(objs)
	if err != nil {
		t.Fatalf("marchallList() failed: %v", err)
	}
	return data
}

func marchallObj(t *testing.T, obj interface{}) []byte {
	data, err := json.Marshal(obj)
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

type httpTest struct {
	name     string
	method   string
	path     string
	body     []byte
	token    string
	wantCode int
	wantData []byte
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

func Test_userApi_userQuery(t *testing.T) {
	e, repo := setup(t)

	path := func(search string, createdFrom, createdTo time.Time, isActive *bool, roles ...string) string {
		v := make(url.Values)
		if search != "" {
			v.Add("search", search)
		}
		if isActive != nil {
			v.Add("is_active", strconv.FormatBool(*isActive))
		}
		if !createdFrom.IsZero() {
			v.Add("created_from", createdFrom.Format(time.RFC3339))
		}
		if !createdTo.IsZero() {
			v.Add("created_to", createdTo.Format(time.RFC3339))
		}
		for _, r := range roles {
			v.Add("role", r)
		}
		return "/v1/users?" + v.Encode()
	}
	bPtr := func(b bool) *bool { return &b }

	now := time.Now()
	t1 := now.Add(1 * time.Hour)
	t2 := now.Add(2 * time.Hour)
	t3 := now.Add(3 * time.Hour)
	t4 := now.Add(4 * time.Hour)
	t5 := now.Add(5 * time.Hour)

	usr1 := createUser(t, repo, "User", "awe", "awe@test.cd", "", nil, true, t1)
	usr2 := createUser(t, repo, "King", "user02", "king@test.cd", "", nil, true)
	student := createUser(t, repo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)
	admin := createUser(t, repo, "Admin", "admin", "admin@test.cd", "", []string{user.RoleAdmin}, true, t2.Truncate(time.Second))
	principal := createUser(t, repo, "Principal", "princip", "princip@test.cd", "", []string{user.RoleAdminPrincipal}, true)
	teacher := createUser(t, repo, "Teacher", "teacher", "teacher@test.cd", "", []string{user.RoleTeacher}, true, t3)
	naughty := createUser(t, repo, "N Dog", "ndog", "ndog@test.cd", "", []string{user.RoleStudent}, false) // 😂

	adminToken := getToken(t, admin)
	empty := marchallList(t)

	tests := []httpTest{
		{name: "Auth required", path: "/v1/users", wantCode: http.StatusUnauthorized, wantData: marchallObj(t, errMissingToken)},
		{name: "Admin required", path: "/v1/users", token: getToken(t, student), wantCode: http.StatusForbidden, wantData: marchallObj(t, httpErr{Error: "permission denied"})},
		{name: "Get all", path: "/v1/users", token: adminToken, wantData: marchallList(t, usr1, usr2, student, admin, principal, teacher, naughty)},
		{name: "search (unknown)", path: path("lol", time.Time{}, time.Time{}, nil), token: adminToken, wantData: empty},
		{name: "search=USE", path: path("USE", time.Time{}, time.Time{}, nil), token: adminToken, wantData: marchallList(t, usr1, usr2, student)},
		{name: "role (unknown)", path: path("", time.Time{}, time.Time{}, nil, "lol"), token: adminToken, wantData: empty},
		{name: "role=admin:", path: path("", time.Time{}, time.Time{}, nil, user.RoleAdmin), token: adminToken, wantData: marchallList(t, admin, principal)},
		{name: "role=teacher:", path: path("", time.Time{}, time.Time{}, nil, user.RoleTeacher), token: adminToken, wantData: marchallList(t, teacher)},
		{name: "role=teacher:,student:", path: path("", time.Time{}, time.Time{}, nil, user.RoleTeacher, user.RoleStudent), token: adminToken, wantData: marchallList(t, teacher, student, naughty)},
		{name: "is_active=true", path: path("", time.Time{}, time.Time{}, bPtr(true)), token: adminToken, wantData: marchallList(t, usr1, usr2, student, admin, principal, teacher)},
		{name: "is_active=false", path: path("", time.Time{}, time.Time{}, bPtr(false)), token: adminToken, wantData: marchallList(t, naughty)},
		{name: "created_from (UTC)", path: path("", t1.UTC(), time.Time{}, nil), token: adminToken, wantData: marchallList(t, usr1, admin, teacher)},
		{name: "created_from (curr TZ)", path: path("", t1, time.Time{}, nil), token: adminToken, wantData: marchallList(t, usr1, admin, teacher)},
		{name: "created_to (curr TZ)", path: path("", time.Time{}, t2, nil), token: adminToken, wantData: marchallList(t, usr1, usr2, student, admin, principal, naughty)},
		{name: "created_from - created_to (empty)", path: path("", t4, t5, nil), token: adminToken, wantData: empty},
		{name: "created_from - created_to (found)", path: path("", t1, t2, nil), token: adminToken, wantData: marchallList(t, usr1, admin)},
		{name: "all combo (empty)", path: path("USE", t1, t5, bPtr(true), user.RoleAdminPrincipal), token: adminToken, wantData: empty},
		{name: "all combo (found)", path: path("tea", t1, t5, bPtr(true), user.RoleTeacher), token: adminToken, wantData: marchallList(t, teacher)},
	}
	for _, tt := range tests {
		tt.method = http.MethodGet
		if tt.wantCode == 0 {
			tt.wantCode = http.StatusOK
		}

		t.Run(tt.name, func(t *testing.T) {
			req, rec := newAuthRequest(tt.method, tt.path, tt.token, tt.body)
			e.ServeHTTP(rec, req)
			defer func() { _ = e.Close() }()
			checkCodeAndData(t, tt, rec)
		})
	}
}

func Test_userApi_userRefreshToken(t *testing.T) {
	e, repo := setup(t)

	naughty := createUser(t, repo, "N Dog", "ndog", "ndog@test.cd", "", []string{user.RoleStudent}, false) // 😂
	student := createUser(t, repo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)

	now := time.Now()
	unrefreshableClaims := &helpers.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "Masomo",
			Subject:   strconv.Itoa(student.ID),
			Audience:  "Academia",
			ExpiresAt: now.Add(helpers.ExpirationDelta).Unix(),
			IssuedAt:  now.Unix(),
		},
		OriginalIssuedAt: now.Add(-2 * helpers.RefreshExpirationDelta).Unix(), // older than threshold
		IsStudent:        student.IsStudent(),
		IsTeacher:        student.IsTeacher(),
		IsAdmin:          student.IsAdmin(),
		Roles:            student.Roles,
	}
	unrefreshableToken, err := helpers.GenerateToken(unrefreshableClaims)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	tests := []httpTest{
		{name: "Auth required", wantCode: http.StatusUnauthorized, wantData: marchallObj(t, errMissingToken)},
		{name: "Inactive user not allowed", token: getToken(t, naughty), wantCode: http.StatusForbidden, wantData: marchallObj(t, httpErr{Error: "account deactivated"})},
		{name: "Refresh period expired", token: unrefreshableToken, wantCode: http.StatusForbidden, wantData: marchallObj(t, httpErr{Error: "refresh has expired"})},
		{name: "Token refreshed", token: getToken(t, student), wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		tt.method = http.MethodPost
		tt.path = "/v1/users/token-refresh"

		t.Run(tt.name, func(t *testing.T) {
			req, rec := newAuthRequest(tt.method, tt.path, tt.token, tt.body)
			e.ServeHTTP(rec, req)
			defer func() { _ = e.Close() }()

			// cannot guess new token.. just check that it's not empty
			if tt.name == "Token refreshed" {
				if rec.Code != tt.wantCode {
					t.Errorf("failed! code = %v; wantCode %v", rec.Code, tt.wantCode)
				}
				var respData LoginResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &respData); err != nil {
					t.Errorf("json.Unmarshal() failed! err %v", err)
				}
				if respData.Token == "" {
					t.Error("failed! empty token")
				}
				return
			}
			checkCodeAndData(t, tt, rec)
		})
	}
}
