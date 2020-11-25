package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
)

func setup(t *testing.T) (*user.Service, user.Repository) {
	db, err := dummydb.Open()
	if err != nil {
		t.Fatalf("setup() failed: %v", err)
	}
	repo := dummydb.NewUserRepository(db)
	svc := user.NewService(repo)
	return svc, repo
}

func newRequest(e *echo.Echo, method, path string, data ...[]byte) (echo.Context, *httptest.ResponseRecorder) {
	var body bytes.Buffer
	if len(data) > 0 {
		body.Write(data[0])
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	return ctx, rec
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

func marchallUsers(t *testing.T, users ...user.User) []byte {
	data, err := json.Marshal(users)
	if err != nil {
		t.Fatalf("marchallUsers() failed: %v", err)
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
	return assert.ElementsMatch(t, j1, j2), nil
}

type httpTest struct {
	name     string
	method   string
	path     string
	body     []byte
	wantCode int
	wantData []byte
	wantErr  error
}

func Test_userApi_userQuery(t *testing.T) {
	svc, repo := setup(t)
	api := &userApi{
		service: svc,
	}
	e := echo.New()

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
		return "/users?" + v.Encode()
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
	empty := marchallUsers(t)

	tests := []httpTest{
		{name: "Get all", path: "/users", wantData: marchallUsers(t, usr1, usr2, student, admin, principal, teacher, naughty)},
		{name: "search (unknown)", path: path("lol", time.Time{}, time.Time{}, nil), wantData: empty},
		{name: "search=USE", path: path("USE", time.Time{}, time.Time{}, nil), wantData: marchallUsers(t, usr1, usr2, student)},
		{name: "role (unknown)", path: path("", time.Time{}, time.Time{}, nil, "lol"), wantData: empty},
		{name: "role=admin:", path: path("", time.Time{}, time.Time{}, nil, user.RoleAdmin), wantData: marchallUsers(t, admin, principal)},
		{name: "role=teacher:", path: path("", time.Time{}, time.Time{}, nil, user.RoleTeacher), wantData: marchallUsers(t, teacher)},
		{name: "role=teacher:,student:", path: path("", time.Time{}, time.Time{}, nil, user.RoleTeacher, user.RoleStudent), wantData: marchallUsers(t, teacher, student, naughty)},
		{name: "is_active=true", path: path("", time.Time{}, time.Time{}, bPtr(true)), wantData: marchallUsers(t, usr1, usr2, student, admin, principal, teacher)},
		{name: "is_active=false", path: path("", time.Time{}, time.Time{}, bPtr(false)), wantData: marchallUsers(t, naughty)},
		{name: "created_from (UTC)", path: path("", t1.UTC(), time.Time{}, nil), wantData: marchallUsers(t, usr1, admin, teacher)},
		{name: "created_from (curr TZ)", path: path("", t1, time.Time{}, nil), wantData: marchallUsers(t, usr1, admin, teacher)},
		{name: "created_to (curr TZ)", path: path("", time.Time{}, t2, nil), wantData: marchallUsers(t, usr1, usr2, student, admin, principal, naughty)},
		{name: "created_from - created_to (empty)", path: path("", t4, t5, nil), wantData: empty},
		{name: "created_from - created_to (found)", path: path("", t1, t2, nil), wantData: marchallUsers(t, usr1, admin)},
		{name: "all combo (empty)", path: path("USE", t1, t5, bPtr(true), user.RoleAdminPrincipal), wantData: empty},
		{name: "all combo (found)", path: path("tea", t1, t5, bPtr(true), user.RoleTeacher), wantData: marchallUsers(t, teacher)},
	}
	for _, tt := range tests {
		tt.method = http.MethodGet
		tt.wantCode = http.StatusOK

		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newRequest(e, tt.method, tt.path, tt.body)
			if err := api.userQuery(ctx); err != tt.wantErr {
				t.Errorf("userQuery() error = %v; wantErr %v", err, tt.wantErr)
			}
			if rec.Code != tt.wantCode {
				t.Errorf("userQuery() code = %v; wantCode %v", rec.Code, tt.wantCode)
			}
			ok, err := jsonBytesEqual(t, rec.Body.Bytes(), tt.wantData)
			if err != nil {
				t.Errorf("jsonBytesEqual() failed to compare; err %v", err)
			}
			if !ok {
				t.Errorf("userQuery() data = %v; wantData %v", rec.Body.String(), string(tt.wantData))
			}
		})
	}
}
