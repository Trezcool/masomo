package testutil

import (
	"testing"
	"time"

	"github.com/trezcool/masomo/backend/core/user"
)

func CreateUser(
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
