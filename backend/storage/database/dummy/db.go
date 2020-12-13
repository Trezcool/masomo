package dummydb

import (
	"sync"

	"github.com/trezcool/masomo/core/user"
)

type (
	DB struct {
		user *userTable
	}

	userTable struct {
		sync.RWMutex
		table map[int]*user.User
	}
)

func Open() (*DB, error) {
	db := &DB{
		user: &userTable{table: make(map[int]*user.User)},
	}
	return db, nil
}
