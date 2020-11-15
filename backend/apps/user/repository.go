package user

import (
	"errors"
	"sync"
	"time"

	"github.com/trezcool/masomo/backend/apps"
	"github.com/trezcool/masomo/backend/apps/utils"
)

var (
	pkCount int

	// errors
	NotFoundErr = errors.New("user not found")
)

type _DB map[int]*User // in-memory; TODO real DB

func (db _DB) query() []User {
	r := make([]User, 0, len(db))
	for _, u := range db {
		r = append(r, *u)
	}
	return r
}

type Repository struct {
	db    _DB
	mutex sync.RWMutex

	//log *log.Logger
}

func NewRepository() *Repository {
	db := make(_DB)

	pkCount++
	now := time.Now()
	root := &User{
		ID:        pkCount,
		Name:      "Root",
		Username:  "root",
		Email:     "root@masomo.cd",
		IsActive:  true,
		Roles:     AllRoles,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = root.SetPassword("LolC@t123")
	db[pkCount] = root

	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(nu NewUser) (User, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	uname := utils.CleanString(nu.Username, true)
	email := utils.CleanString(nu.Email, true)
	if err := r.checkUniqueness(uname, email); err != nil { // TODO: avoid user enumeration
		return User{}, err
	}

	pkCount++
	now := time.Now()
	usr := User{
		ID:        pkCount,
		Name:      utils.CleanString(nu.Name),
		Username:  uname,
		Email:     email,
		IsActive:  true,
		Roles:     nu.Roles,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := usr.SetPassword(nu.Password); err != nil {
		return User{}, err
	}
	r.db[pkCount] = &usr

	return usr, nil
}

func (r *Repository) checkUniqueness(uname, email string) error {
	for _, usr := range r.db.query() {
		if uname != "" && usr.Username == uname {
			return apps.NewArgumentError("a user with this username already exists")
		}
		if email != "" && usr.Email == email {
			return apps.NewArgumentError("a user with this email already exists")
		}
	}
	return nil
}

func (r *Repository) Query(q ...QueryFilter) ([]User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.db.query(), nil
}

func (r *Repository) GetByID(id int) (User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if id != 0 {
		if usr, ok := r.db[id]; ok {
			return *usr, nil
		}
	}
	return User{}, NotFoundErr
}

func (r *Repository) GetByUsername(uname string) (User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	uname = utils.CleanString(uname, true)
	if uname != "" {
		for _, usr := range r.db.query() {
			if usr.Username == uname {
				return usr, nil
			}
		}
	}
	return User{}, NotFoundErr
}

func (r *Repository) GetByEmail(email string) (User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	email = utils.CleanString(email, true)
	if email != "" {
		for _, usr := range r.db.query() {
			if usr.Email == email {
				return usr, nil
			}
		}
	}
	return User{}, NotFoundErr
}

func (r *Repository) GetByUsernameOrEmail(uname string) (User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	uname = utils.CleanString(uname, true)
	if uname != "" {
		for _, usr := range r.db.query() {
			if (usr.Username == uname) || (usr.Email == uname) {
				return usr, nil
			}
		}
	}
	return User{}, NotFoundErr
}

func (r *Repository) Update(id int, uu UpdateUser) (User, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// TODO: unique username & email check (except user)

	return User{}, nil
}

func (r *Repository) Delete(id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.db, id)
	return nil
}
