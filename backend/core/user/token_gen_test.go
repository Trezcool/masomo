package user

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/trezcool/masomo/core"
)

func TestMakeVerifyToken(t *testing.T) {
	now := time.Now()
	usr := User{
		ID:        uuid.New().String(),
		Name:      "T",
		Username:  "t",
		Email:     "t@test.test",
		CreatedAt: now,
		UpdatedAt: now,
		LastLogin: now,
	}
	usr.SetActive(true)
	_ = usr.SetPassword("pwd")

	validToken, err := MakeToken(usr)
	if err != nil {
		t.Fatalf("MakeToken() failed, %v", err)
	}

	// generate an expired token
	dayLate := core.Conf.PasswordResetTimeoutDelta + (24 * time.Hour)
	NowFunc = func() time.Time { return time.Now().Add(-dayLate) }
	expiredToken, err := MakeToken(usr)
	if err != nil {
		t.Fatalf("MakeToken() failed, %v", err)
	}
	NowFunc = time.Now // reset

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
