package user

import (
	"testing"
	"time"
)

func TestMakeVerifyToken(t *testing.T) {
	secretKey = []byte("secret")
	passwordResetTimeoutDelta = 3 * 24 * time.Hour

	now := time.Now()
	usr := User{
		ID:        1,
		Name:      "T",
		Username:  "t",
		Email:     "t@test.test",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
		LastLogin: now,
	}
	_ = usr.SetPassword("pwd")

	validToken := makeToken(usr)

	// generate an expired token
	dayLate := passwordResetTimeoutDelta + (24 * time.Hour)
	nowFunc = func() time.Time { return time.Now().Add(-dayLate) }
	expiredToken := makeToken(usr)
	nowFunc = time.Now // reset

	tests := []struct {
		name    string
		usr     User
		token   string
		wantErr error
	}{
		{name: "no token", usr: usr, wantErr: errInvalidToken},
		{name: "invalid parts len", usr: usr, token: "lmaooolol", wantErr: errInvalidToken},
		{name: "invalid base32", usr: usr, token: "hahaha-sigsig-sig", wantErr: errInvalidToken},
		{name: "invalid timestamp", usr: usr, token: "NRXWY-sigsig-sig", wantErr: errInvalidToken},
		{name: "invalid token", usr: usr, token: "HE4TS-sigsig-sig", wantErr: errInvalidToken},
		{name: "expired token", usr: usr, token: expiredToken, wantErr: errTokenExpired},
		{name: "valid token", usr: usr, token: validToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifyToken(tt.usr, tt.token); err != tt.wantErr {
				t.Errorf("verifyToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
