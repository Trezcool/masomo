package user

import (
	"errors"
	"sync"
	"time"

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
	return &Repository{
		db: make(_DB),
	}
}

func (r *Repository) Create(nu NewUser) (User, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	pkCount++
	now := time.Now()
	usr := User{
		ID:        pkCount,
		Name:      nu.Name,
		Username:  nu.Username,
		Email:     nu.Email,
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
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, usr := range r.db.query() {
		if uname != "" && usr.Username == uname {
			return utils.NewValidationError(nil, utils.FieldError{
				Field: "username",
				Error: "a user with this username already exists",
			})
		}
		if email != "" && usr.Email == email {
			return utils.NewValidationError(nil, utils.FieldError{
				Field: "email",
				Error: "a user with this email already exists",
			})
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
