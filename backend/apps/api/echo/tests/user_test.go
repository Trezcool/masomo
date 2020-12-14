package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/services/email"
	"github.com/trezcool/masomo/tests"
)

func Test_userApi_userQuery(t *testing.T) {
	testutil.ResetDB(t, db)

	path := func(search, ordering string, createdFrom, createdTo time.Time, isActive *bool, roles ...string) string {
		v := make(url.Values)
		if search != "" {
			v.Add("search", search)
		}
		if ordering != "" {
			v.Add("ordering", ordering)
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
		return "/api/users?" + v.Encode()
	}
	bPtr := func(b bool) *bool { return &b }

	now := time.Now()
	t1 := now.Add(1 * time.Hour)
	t2 := now.Add(2 * time.Hour)
	t3 := now.Add(3 * time.Hour)
	t4 := now.Add(4 * time.Hour)
	t5 := now.Add(5 * time.Hour)

	fmt.Printf("\n\nnow: %v", now)
	fmt.Printf("\nnow.T: %v", now.Truncate(time.Microsecond))
	fmt.Printf("\nt1: %v", t1)
	fmt.Printf("\nt2: %v", t2)
	fmt.Printf("\nt3: %v", t3)
	fmt.Printf("\nt3.T: %v", t3.Truncate(time.Microsecond))
	fmt.Printf("\nt4: %v", t4)
	fmt.Printf("\nt5: %v\n\n", t5)

	usr1 := testutil.CreateUser(t, usrRepo, "User", "awe", "awe@test.cd", "", nil, true, t1)
	usr2 := testutil.CreateUser(t, usrRepo, "King", "user02", "king@test.cd", "", nil, true)
	student := testutil.CreateUser(t, usrRepo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)
	admin := testutil.CreateUser(t, usrRepo, "Admin", "admin", "admin@test.cd", "", []string{user.RoleAdmin}, true, t2.Truncate(time.Second))
	principal := testutil.CreateUser(t, usrRepo, "Principal", "princip", "princip@test.cd", "", []string{user.RoleAdminPrincipal}, true)
	teacher := testutil.CreateUser(t, usrRepo, "Teacher", "teacher", "teacher@test.cd", "", []string{user.RoleTeacher}, true, t3)
	naughty := testutil.CreateUser(t, usrRepo, "N Dog", "ndog", "ndog@test.cd", "", []string{user.RoleStudent}, false) // 😂

	adminToken := getToken(t, admin)
	empty := marchallList(t, []interface{}{}...)

	tests := []httpTest{
		{name: "Auth required", path: "/api/users", wantCode: http.StatusUnauthorized, wantData: marchallObj(t, errMissingToken)},
		{
			name: "Admin required", path: "/api/users", token: getToken(t, student), wantCode: http.StatusForbidden,
			wantData: marchallObj(t, httpErr{Error: "permission denied"}),
		},
		{
			name: "Get all", path: "/api/users", token: adminToken,
			wantData: marchallList(t, teacher, admin, usr1, naughty, principal, student, usr2),
		},
		// filtering
		{name: "search (unknown)", path: path("lol", "", time.Time{}, time.Time{}, nil), token: adminToken, wantData: empty},
		{
			name: "search=USE", path: path("USE", "", time.Time{}, time.Time{}, nil),
			token: adminToken, wantData: marchallList(t, usr1, student, usr2),
		},
		{name: "role (unknown)", path: path("", "", time.Time{}, time.Time{}, nil, "lol"), token: adminToken, wantData: empty},
		{
			name: "role=admin:", path: path("", "", time.Time{}, time.Time{}, nil, user.RoleAdmin),
			token: adminToken, wantData: marchallList(t, admin, principal),
		},
		{
			name: "role=teacher:", path: path("", "", time.Time{}, time.Time{}, nil, user.RoleTeacher),
			token: adminToken, wantData: marchallList(t, teacher),
		},
		{
			name: "role=teacher:,student:", path: path("", "", time.Time{}, time.Time{}, nil, user.RoleTeacher, user.RoleStudent),
			token: adminToken, wantData: marchallList(t, teacher, naughty, student),
		},
		{
			name: "is_active=true", path: path("", "", time.Time{}, time.Time{}, bPtr(true)),
			token: adminToken, wantData: marchallList(t, teacher, admin, usr1, principal, student, usr2),
		},
		{name: "is_active=false", path: path("", "", time.Time{}, time.Time{}, bPtr(false)), token: adminToken, wantData: marchallList(t, naughty)},
		{
			name: "created_from (UTC)", path: path("", "", t1.UTC(), time.Time{}, nil),
			token: adminToken, wantData: marchallList(t, teacher, admin, usr1),
		},
		{
			name: "created_from (curr TZ)", path: path("", "", t1, time.Time{}, nil),
			token: adminToken, wantData: marchallList(t, teacher, admin, usr1),
		},
		{
			name: "created_to (curr TZ)", path: path("", "", time.Time{}, t2, nil),
			token: adminToken, wantData: marchallList(t, admin, usr1, naughty, principal, student, usr2),
		},
		{name: "created_from - created_to (empty)", path: path("", "", t4, t5, nil), token: adminToken, wantData: empty},
		{name: "created_from - created_to (found)", path: path("", "", t1, t2, nil), token: adminToken, wantData: marchallList(t, admin, usr1)},
		{name: "all combo (empty)", path: path("USE", "", t1, t5, bPtr(true), user.RoleAdminPrincipal), token: adminToken, wantData: empty},
		{
			name: "all combo (found)", path: path("tea", "", t1, t5, bPtr(true), user.RoleTeacher),
			token: adminToken, wantData: marchallList(t, teacher),
		},
		// ordering
		{
			name: "order by created_at", path: path("", "created_at", time.Time{}, time.Time{}, nil), token: adminToken,
			wantData: marchallList(t, usr2, student, principal, naughty, usr1, admin, teacher),
		},
		{
			name: "order by -created_at", path: path("", "-created_at", time.Time{}, time.Time{}, nil), token: adminToken,
			wantData: marchallList(t, teacher, admin, usr1, naughty, principal, student, usr2),
		},
		{
			name: "order by is_active,-name", path: path("", "is_active,-name", time.Time{}, time.Time{}, nil), token: adminToken,
			wantData: marchallList(t, naughty, usr1, teacher, principal, usr2, student, admin),
		},
		{
			name: "order by -is_active,name", path: path("", "-is_active,name", time.Time{}, time.Time{}, nil), token: adminToken,
			wantData: marchallList(t, admin, student, usr2, principal, teacher, usr1, naughty),
		},
		// filtering & ordering
		{
			name: "filtering & ordering", path: path("", "name", time.Time{}, time.Time{}, nil, user.RoleTeacher, user.RoleStudent), token: adminToken,
			wantData: marchallList(t, student, naughty, teacher),
		},
	}
	for _, tt := range tests {
		tt.method = http.MethodGet
		if tt.wantCode == 0 {
			tt.wantCode = http.StatusOK
		}

		t.Run(tt.name, func(t *testing.T) {
			req, rec := newAuthRequest(tt.method, tt.path, tt.token, tt.body)
			app.ServeHTTP(rec, req)
			checkCodeAndData(t, tt, rec)
		})
	}
}

func Test_userApi_userRefreshToken(t *testing.T) {
	testutil.ResetDB(t, db)

	naughty := testutil.CreateUser(t, usrRepo, "N Dog", "ndog", "ndog@test.cd", "", []string{user.RoleStudent}, false) // 😂
	student := testutil.CreateUser(t, usrRepo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)

	now := time.Now()
	unrefreshableClaims := &echoapi.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "Masomo",
			Subject:   student.ID,
			Audience:  "Academia",
			ExpiresAt: now.Add(core.Conf.JWTExpirationDelta).Unix(),
			IssuedAt:  now.Unix(),
		},
		OrigIssuedAt: now.Add(-2 * core.Conf.JWTRefreshExpirationDelta).Unix(), // older than threshold
		IsStudent:    student.IsStudent(),
		IsTeacher:    student.IsTeacher(),
		IsAdmin:      student.IsAdmin(),
		Roles:        student.Roles,
	}
	unrefreshableToken, err := echoapi.GenerateToken(unrefreshableClaims)
	if err != nil {
		t.Fatalf("GenerateToken(): %v", err)
	}

	tests := []httpTest{
		{name: "Auth required", wantCode: http.StatusUnauthorized, wantData: marchallObj(t, errMissingToken)},
		{name: "Inactive user not allowed", token: getToken(t, naughty), wantCode: http.StatusForbidden, wantData: marchallObj(t, httpErr{Error: "account deactivated"})},
		{name: "Refresh period expired", token: unrefreshableToken, wantCode: http.StatusForbidden, wantData: marchallObj(t, httpErr{Error: "refresh has expired"})},
		{name: "Token refreshed", token: getToken(t, student), wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		tt.method = http.MethodPost
		tt.path = "/api/users/token-refresh"

		t.Run(tt.name, func(t *testing.T) {
			req, rec := newAuthRequest(tt.method, tt.path, tt.token, tt.body)
			app.ServeHTTP(rec, req)

			// cannot guess new token.. just check that it's not empty
			if tt.wantCode == http.StatusOK {
				if rec.Code != tt.wantCode {
					t.Errorf("failed! code = %v; wantCode %v", rec.Code, tt.wantCode)
				}
				var respData echoapi.LoginResponse
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
	testutil.ResetDB(t, db)

	student := testutil.CreateUser(t, usrRepo, "Hero", "hero", "user3@test.cd", "", []string{user.RoleStudent}, true)
	successData := marchallObj(t, echoapi.SuccessResponse{Success: "If the email address supplied is associated with an active account on this system, " +
		"an email will arrive in your inbox shortly with instructions to reset your password."})

	pathRegex, err := regexp.Compile("/password-reset/.+/.+")
	if err != nil {
		t.Fatalf("regexp.Compile(): %v", err)
	}

	type extraTest struct {
		emailSent bool
		to        mail.Address
	}
	tests := []httpTest{
		{name: "required fields", wantCode: http.StatusBadRequest, wantData: marchallObj(t, echoapi.PasswordResetRequest{Email: "this field is required"})},
		{
			name: "invalid email", wantCode: http.StatusBadRequest, body: marchallObj(t, echoapi.PasswordResetRequest{Email: "lol"}),
			wantData: marchallObj(t, echoapi.PasswordResetRequest{Email: "email must be a valid email address"}),
		},
		{
			name: "unknown email", wantCode: http.StatusOK, body: marchallObj(t, echoapi.PasswordResetRequest{Email: "lol@test.com"}),
			wantData: successData, extra: extraTest{emailSent: false},
		},
		{
			name: "know email", wantCode: http.StatusOK, body: marchallObj(t, echoapi.PasswordResetRequest{Email: student.Email}),
			wantData: successData, extra: extraTest{emailSent: true, to: mail.Address{Name: student.Name, Address: student.Email}},
		},
	}
	for _, tt := range tests {
		tt.method = http.MethodPost
		tt.path = "/api/users/password-reset"

		t.Run(tt.name, func(t *testing.T) {
			emailsvc.SentMessages = nil // reset

			req, rec := newRequest(tt.method, tt.path, tt.body)
			app.ServeHTTP(rec, req)
			checkCodeAndData(t, tt, rec)

			if extra, ok := tt.extra.(extraTest); ok {
				if extra.emailSent {
					if len(emailsvc.SentMessages) != 1 {
						t.Fatalf("failed! len(SentMessages) = %d; want 1", len(emailsvc.SentMessages))
					}
					msg := emailsvc.SentMessages[0]
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
					if len(emailsvc.SentMessages) > 0 {
						t.Errorf("failed! len(SentMessages) = %d; want 0", len(emailsvc.SentMessages))
					}
				}
			}
		})
	}
}

func Test_userApi_userConfirmPasswordReset(t *testing.T) {
	testutil.ResetDB(t, db)

	student := testutil.CreateUser(t, usrRepo, "Hero", "hero", "user3@test.cd", "lol", []string{user.RoleStudent}, true)
	validUID := user.EncodeUID(student)
	validToken, err := user.MakeToken(student)
	if err != nil {
		t.Fatalf("MakeToken(): %v", err)
	}

	// generate an expired token
	dayLate := core.Conf.PasswordResetTimeoutDelta + (24 * time.Hour)
	user.NowFunc = func() time.Time { return time.Now().Add(-dayLate) }
	expiredToken, err := user.MakeToken(student)
	if err != nil {
		t.Fatalf("MakeToken(): %v", err)
	}
	user.NowFunc = time.Now // reset

	reqMsg := "this field is required"
	tests := []httpTest{
		{
			name: "required fields", wantCode: http.StatusBadRequest,
			wantData: marchallObj(t, user.ResetUserPassword{Token: reqMsg, UID: reqMsg, Password: "password must contain at least 8 characters", PasswordConfirm: reqMsg}),
		},
		{
			name: "invalid pwd: min len", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "lol", Password: "lol", PasswordConfirm: "lol"}),
			wantData: marchallObj(t, user.ResetUserPassword{Password: "password must contain at least 8 characters"}),
		},
		{
			name: "invalid pwd: no whitespace", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "lol", Password: "l o loll", PasswordConfirm: "l o loll"}),
			wantData: marchallObj(t, user.ResetUserPassword{Password: "password must not contain whitespace"}),
		},
		{
			name: "invalid pwd: not all numeric", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "lol", Password: "12345678", PasswordConfirm: "12345678"}),
			wantData: marchallObj(t, user.ResetUserPassword{Password: "password cannot be entirely numeric"}),
		},
		{
			name: "invalid pwd: complexity", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "lol", Password: "lol12345", PasswordConfirm: "lol12345"}),
			wantData: marchallObj(t, user.ResetUserPassword{Password: "password must contain at least 1 uppercase character, 1 lowercase character, 1 digit and 1 special character"}),
		},
		{
			name: "invalid pwd: too common", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "lol", Password: "P@$$w0rd", PasswordConfirm: "P@$$w0rd"}),
			wantData: marchallObj(t, user.ResetUserPassword{Password: "password is too common"}),
		},
		{
			name: "PasswordConfirm must = Password", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "lol", Password: "LolC@t123", PasswordConfirm: "lol"}),
			wantData: marchallObj(t, user.ResetUserPassword{PasswordConfirm: "password_confirm must be equal to Password"}),
		},
		{
			name: "invalid uid", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "bG9s", Password: "LolC@t123", PasswordConfirm: "LolC@t123"}),
			wantData: marchallObj(t, user.ResetUserPassword{UID: "invalid value"}),
		},
		{
			name: "user not found", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "lol", UID: "OTk5", Password: "LolC@t123", PasswordConfirm: "LolC@t123"}),
			wantData: marchallObj(t, user.ResetUserPassword{UID: "invalid value"}),
		},
		{
			name: "invalid token", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: "HE4TS-sigsig-sig", UID: validUID, Password: "LolC@t123", PasswordConfirm: "LolC@t123"}),
			wantData: marchallObj(t, user.ResetUserPassword{Token: "invalid value"}),
		},
		{
			name: "expired token", wantCode: http.StatusBadRequest,
			body:     marchallObj(t, user.ResetUserPassword{Token: expiredToken, UID: validUID, Password: "LolC@t123", PasswordConfirm: "LolC@t123"}),
			wantData: marchallObj(t, user.ResetUserPassword{Token: "invalid value"}),
		},
		{
			name: "valid token", wantCode: http.StatusOK,
			body:     marchallObj(t, user.ResetUserPassword{Token: validToken, UID: validUID, Password: "LolC@t123", PasswordConfirm: "LolC@t123"}),
			wantData: marchallObj(t, echoapi.SuccessResponse{Success: "Password has been reset with the new password."}),
		},
	}
	for _, tt := range tests {
		tt.method = http.MethodPost
		tt.path = "/api/users/password-reset-confirm"

		t.Run(tt.name, func(t *testing.T) {
			req, rec := newRequest(tt.method, tt.path, tt.body)
			app.ServeHTTP(rec, req)
			checkCodeAndData(t, tt, rec)

			if tt.wantCode == http.StatusOK {
				refreshedStudent, err := usrRepo.GetUser(context.Background(), user.GetFilter{ID: student.ID})
				if err != nil {
					t.Fatalf("GetUserByID() failed, %v", err)
				}
				if bytes.Equal(refreshedStudent.PasswordHash, student.PasswordHash) {
					t.Fatalf("failed to update new password")
				}
			}
		})
	}
}
