package user

import (
	"errors"
	"time"

	"github.com/trezcool/masomo/backend/apps/utils"
)

var (
	// errors
	NotFoundErr       = errors.New("user not found")
	EmailExistsErr    = errors.New("a user with this email already exists")
	UsernameExistsErr = errors.New("a user with this username already exists")
)

type (
	Repository interface {
		CheckUsernameUniqueness(username, email string) error
		CreateUser(user User) (User, error)
		QueryAllUsers() ([]User, error)
		GetUserByID(id int) (User, error)
		GetUserByUsername(username string) (User, error)
		GetUserByEmail(email string) (User, error)
		GetUserByUsernameOrEmail(username string) (User, error)
		UpdateUser(user User) (User, error)
		DeleteUser(id int) error
	}

	Service struct {
		r Repository
		//log *log.Logger
	}
)

func NewService(repo Repository) *Service {
	return &Service{r: repo}
}

func (s *Service) checkUniqueness(uname, email string) error {
	if err := s.r.CheckUsernameUniqueness(uname, email); err != nil {
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

func (s *Service) Create(nu NewUser) (User, error) {
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
	return s.r.CreateUser(usr)
}

func (s *Service) QueryAll() ([]User, error) {
	return s.r.QueryAllUsers()
}

func (s *Service) GetByID(id int) (User, error) {
	return s.r.GetUserByID(id)
}

func (s *Service) GetByUsername(uname string) (User, error) {
	return s.r.GetUserByUsername(utils.CleanString(uname, true))
}

func (s *Service) GetByEmail(email string) (User, error) {
	return s.r.GetUserByEmail(utils.CleanString(email, true))
}

func (s *Service) GetByUsernameOrEmail(uname string) (User, error) {
	return s.r.GetUserByUsernameOrEmail(utils.CleanString(uname, true))
}

func (s *Service) Update(id int, uu UpdateUser) (User, error) {
	// TODO: unique username & email check (except user)
	return User{}, nil
}

func (s *Service) Delete(id int) error {
	return s.r.DeleteUser(id)
}
