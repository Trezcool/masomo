package user

import (
	"errors"
	"time"

	"github.com/trezcool/masomo/backend/core"
)

var (
	// errors
	ErrNotFound       = errors.New("user not found")
	ErrEmailExists    = errors.New("a user with this email already exists")
	ErrUsernameExists = errors.New("a user with this username already exists")
)

type (
	Repository interface {
		CheckUsernameUniqueness(username, email string, excludedUsers ...User) error
		CreateUser(user User) (User, error)
		QueryAllUsers() ([]User, error)
		GetUserByID(id int) (User, error)
		GetUserByUsername(username string) (User, error)
		GetUserByEmail(email string) (User, error)
		GetUserByUsernameOrEmail(username string) (User, error)
		// FilterUsers applies AND operation on available QueryFilter fields.
		// QueryFilter.Search does a case-insensitive match on one of User.Name, User.Username or User.Email.
		FilterUsers(filter QueryFilter) ([]User, error) // TODO: make it functional ?
		UpdateUser(user User, isActive *bool) (User, error)
		DeleteUsersByID(ids ...int) error
	}

	Service struct {
		repo Repository
		//log *log.Logger
	}
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (svc *Service) checkUniqueness(uname, email string, exclUsers ...User) error {
	if err := svc.repo.CheckUsernameUniqueness(uname, email, exclUsers...); err != nil {
		var field string
		switch err {
		case ErrUsernameExists:
			field = "username"
		case ErrEmailExists:
			field = "email"
		default:
			return err
		}
		return core.NewValidationError(err, core.FieldError{Field: field, Error: err.Error()})
	}
	return nil
}

func (svc *Service) Create(nu NewUser) (User, error) {
	now := time.Now().UTC()
	usr := User{
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
	return svc.repo.CreateUser(usr)
}

func (svc *Service) QueryAll() ([]User, error) {
	return svc.repo.QueryAllUsers()
}

func (svc *Service) GetByID(id int) (User, error) {
	return svc.repo.GetUserByID(id)
}

func (svc *Service) GetByUsername(uname string) (User, error) {
	return svc.repo.GetUserByUsername(core.CleanString(uname, true /* lower */))
}

func (svc *Service) GetByEmail(email string) (User, error) {
	return svc.repo.GetUserByEmail(core.CleanString(email, true /* lower */))
}

func (svc *Service) GetByUsernameOrEmail(uname string) (User, error) {
	return svc.repo.GetUserByUsernameOrEmail(core.CleanString(uname, true /* lower */))
}

func (svc *Service) Filter(filter QueryFilter) ([]User, error) {
	return svc.repo.FilterUsers(filter)
}

func (svc *Service) Update(id int, uu UpdateUser) (User, error) {
	usr := User{
		ID:        id,
		Name:      uu.Name,
		Username:  uu.Username,
		Email:     uu.Email,
		Roles:     uu.Roles,
		UpdatedAt: time.Now().UTC(),
	}
	if uu.Password != "" {
		if err := usr.SetPassword(uu.Password); err != nil {
			return User{}, err
		}
	}
	return svc.repo.UpdateUser(usr, uu.IsActive)
}

func (svc *Service) Delete(ids ...int) error {
	return svc.repo.DeleteUsersByID(ids...)
}
