package in_memdb

import (
	"sync"

	"github.com/trezcool/masomo/backend/apps/user"
)

type (
	DB struct {
		user *userTable
	}

	userTable struct {
		t     map[int]*user.User
		mutex sync.RWMutex
	}
)

func Open() (*DB, error) {
	db := &DB{
		user: &userTable{t: make(map[int]*user.User)},
	}
	return db, nil
}
