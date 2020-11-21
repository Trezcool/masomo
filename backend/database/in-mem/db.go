package in_memdb

import (
	"sync"

	"github.com/trezcool/masomo/backend/business/user"
)

type (
	DB struct {
		user *userTable
	}

	userTable struct {
		table map[int]*user.User
		mutex sync.RWMutex
	}
)

func Open() (*DB, error) {
	db := &DB{
		user: &userTable{table: make(map[int]*user.User)},
	}
	return db, nil
}
