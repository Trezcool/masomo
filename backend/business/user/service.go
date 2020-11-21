package user

import (
	"errors"
	"time"

	"github.com/trezcool/masomo/backend/business/utils"
)

var (
	// errors
	NotFoundErr       = errors.New("user not found")
	EmailExistsErr    = errors.New("a user with this email already exists")
	UsernameExistsErr = errors.New("a user with this username already exists")
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
		UpdateUser(user User, isActive *bool) (User, error)
		DeleteUser(id int) error
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
		switch err {
		case UsernameExistsErr:
			return utils.NewValidationError(err, utils.FieldError{Field: "username", Error: err.Error()})
		case EmailExistsErr:
			return utils.NewValidationError(err, utils.FieldError{Field: "email", Error: err.Error()})
		default:
			return err
		}
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
	return svc.repo.GetUserByUsername(utils.CleanString(uname, true))
}

func (svc *Service) GetByEmail(email string) (User, error) {
	return svc.repo.GetUserByEmail(utils.CleanString(email, true))
}

func (svc *Service) GetByUsernameOrEmail(uname string) (User, error) {
	return svc.repo.GetUserByUsernameOrEmail(utils.CleanString(uname, true))
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

func (svc *Service) Delete(id int) error {
	return svc.repo.DeleteUser(id)
}
