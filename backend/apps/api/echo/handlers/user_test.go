package handlers

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/backend/apps/api/echo/helpers"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
	"github.com/trezcool/masomo/backend/tests"
)

func setup(t *testing.T) (*echo.Echo, user.Repository) {
	db, err := dummydb.Open()
	if err != nil {
		t.Fatalf("setup() failed: %v", err)
	}
	repo := dummydb.NewUserRepository(db)
	mailSvc := dummymail.NewService(appName, defaultFromEmail)
	svc := user.NewService(repo, mailSvc, secretKey, passwordResetTimeoutDelta)
	app, v1, jwtMid := initApp()
	RegisterUserAPI(v1, jwtMid, svc)
	return app, repo
}

func Test_userApi_userQuery(t *testing.T) {
	app, repo := setup(t)

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

	usr1 := testutil.CreateUser(t, repo, "User", "awe", "awe@test.cd", "", nil, true, t1)
	usr2 := testutil.CreateUser(t, repo, "King", "user02", "king@test.cd", "", nil, true)
	student := testutil.CreateUser(t, repo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)
	admin := testutil.CreateUser(t, repo, "Admin", "admin", "admin@test.cd", "", []string{user.RoleAdmin}, true, t2.Truncate(time.Second))
	principal := testutil.CreateUser(t, repo, "Principal", "princip", "princip@test.cd", "", []string{user.RoleAdminPrincipal}, true)
	teacher := testutil.CreateUser(t, repo, "Teacher", "teacher", "teacher@test.cd", "", []string{user.RoleTeacher}, true, t3)
	naughty := testutil.CreateUser(t, repo, "N Dog", "ndog", "ndog@test.cd", "", []string{user.RoleStudent}, false) // 😂

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
			app.ServeHTTP(rec, req)
			defer app.Close()
			checkCodeAndData(t, tt, rec)
		})
	}
}

func Test_userApi_userRefreshToken(t *testing.T) {
	app, repo := setup(t)

	naughty := testutil.CreateUser(t, repo, "N Dog", "ndog", "ndog@test.cd", "", []string{user.RoleStudent}, false) // 😂
	student := testutil.CreateUser(t, repo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)

	now := time.Now()
	unrefreshableClaims := &helpers.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "Masomo",
			Subject:   strconv.Itoa(student.ID),
			Audience:  "Academia",
			ExpiresAt: now.Add(jwtExpirationDelta).Unix(),
			IssuedAt:  now.Unix(),
		},
		OriginalIssuedAt: now.Add(-2 * jwtRefreshExpirationDelta).Unix(), // older than threshold
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
			app.ServeHTTP(rec, req)
			defer app.Close()

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

func Test_userApi_userResetPassword(t *testing.T) {
	app, _ := setup(t)

	//student := testutil.CreateUser(t, repo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)
	successData := marchallObj(t, SuccessResponse{Success: "If the email address supplied is associated with an active account on this system, " +
		"an email will arrive in your inbox shortly with instructions to reset your password."})

	pathRegex, err := regexp.Compile("/password-reset/.+/.+")
	if err != nil {
		t.Errorf("pathRegex failed, %v", err)
	}

	type extraTest struct {
		emailSent bool
		to        mail.Address
	}
	tests := []httpTest{
		{name: "required fields", wantCode: http.StatusBadRequest, wantData: marchallObj(t, PasswordResetRequest{Email: "this field is required"})},
		{name: "invalid email", wantCode: http.StatusBadRequest, body: marchallObj(t, PasswordResetRequest{Email: "lol"}),
			wantData: marchallObj(t, PasswordResetRequest{Email: "email must be a valid email address"})},
		{name: "unknown email", wantCode: http.StatusOK, body: marchallObj(t, PasswordResetRequest{Email: "lol@test.com"}), wantData: successData,
			extra: extraTest{emailSent: false}},
		// fixme...
		//{name: "know email", wantCode: http.StatusOK, body: marchallObj(t, PasswordResetRequest{Email: student.Email}), wantData: successData,
		//	extra: extraTest{emailSent: true, to: mail.Address{Name: student.Name, Address: student.Email}}},
	}
	for _, tt := range tests {
		tt.method = http.MethodPost
		tt.path = "/v1/users/password-reset"

		t.Run(tt.name, func(t *testing.T) {
			dummymail.SentMessages = nil // reset

			req, rec := newRequest(tt.method, tt.path, tt.body)
			app.ServeHTTP(rec, req)
			defer app.Close()
			checkCodeAndData(t, tt, rec)

			if extra, ok := tt.extra.(extraTest); ok {
				if extra.emailSent {
					if len(dummymail.SentMessages) != 1 {
						t.Errorf("failed! len(SentMessages) = %d; want 1", len(dummymail.SentMessages))
					}
					msg := dummymail.SentMessages[0]
					if msg.To[0] != extra.to {
						t.Errorf("failed! To = %v; want %v", msg.To[0], extra.to)
					}
					if !strings.Contains(msg.TextContent, extra.to.Name) {
						t.Errorf("failed! text content does not contain recipient's name \"%s\"", extra.to.Name)
					}
					if !strings.Contains(msg.HTMLContent, extra.to.Name) {
						t.Errorf("failed! HTML content does not contain recipient's name \"%s\"", extra.to.Name)
					}
					if !pathRegex.MatchString(msg.TextContent) {
						t.Errorf("failed! text content does not match pathRegex %v", pathRegex)
					}
					if !pathRegex.MatchString(msg.HTMLContent) {
						t.Errorf("failed! HTML content does not match pathRegex %v", pathRegex)
					}
				} else {
					if len(dummymail.SentMessages) > 0 {
						t.Errorf("failed! len(SentMessages) = %d; want 0", len(dummymail.SentMessages))
					}
				}
			}
		})
	}
}
