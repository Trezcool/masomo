package user

import (
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/trezcool/masomo/backend/core"
)

var (
	secretKey                 []byte
	passwordResetTimeoutDelta time.Duration

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
		UpdateUser(user User, isActive ...*bool) (User, error)
		DeleteUsersByID(ids ...int) error
	}

	Service interface {
		CheckUniqueness(uname, email string, exclUsers ...User) error
		Create(nu NewUser) (User, error)
		QueryAll() ([]User, error)
		GetByID(id int) (User, error)
		GetByUsername(uname string) (User, error)
		GetByEmail(email string) (User, error)
		GetByUsernameOrEmail(uname string) (User, error)
		Filter(filter QueryFilter) ([]User, error)
		Update(id int, uu UpdateUser) (User, error)
		SetLastLogin(usr User) (User, error)
		RequestPasswordReset(email string) error
		Delete(ids ...int) error
	}

	service struct {
		repo    Repository
		mailSvc core.EmailService
		//log *log.Logger
	}
)

var _ Service = (*service)(nil)

// todo: pass ctx (with `User` or `isAdmin`) to all methods for permission checks...
func NewService(repo Repository, mailSvc core.EmailService, secret []byte, pwdResetTimeout time.Duration) Service {
	secretKey = secret
	passwordResetTimeoutDelta = pwdResetTimeout
	return &service{
		repo:    repo,
		mailSvc: mailSvc,
	}
}

func (svc *service) CheckUniqueness(uname, email string, exclUsers ...User) error {
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

func (svc *service) Create(nu NewUser) (User, error) {
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

func (svc *service) QueryAll() ([]User, error) {
	return svc.repo.QueryAllUsers()
}

func (svc *service) GetByID(id int) (User, error) {
	return svc.repo.GetUserByID(id)
}

func (svc *service) GetByUsername(uname string) (User, error) {
	return svc.repo.GetUserByUsername(core.CleanString(uname, true /* lower */))
}

func (svc *service) GetByEmail(email string) (User, error) {
	return svc.repo.GetUserByEmail(core.CleanString(email, true /* lower */))
}

func (svc *service) GetByUsernameOrEmail(uname string) (User, error) {
	return svc.repo.GetUserByUsernameOrEmail(core.CleanString(uname, true /* lower */))
}

func (svc *service) Filter(filter QueryFilter) ([]User, error) {
	return svc.repo.FilterUsers(filter)
}

func (svc *service) Update(id int, uu UpdateUser) (User, error) {
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

func (svc *service) SetLastLogin(usr User) (User, error) {
	usr.LastLogin = time.Now().UTC()
	return svc.repo.UpdateUser(usr)
}

func (svc *service) RequestPasswordReset(email string) error {
	usr, err := svc.repo.GetUserByEmail(email)
	if err != nil {
		return err
	}
	// do not wait for it; avoid giving clues to attackers
	go svc.sendPasswordResetMail(usr)
	return nil
}

func (svc *service) sendPasswordResetMail(usr User) {
	token, err := makeToken(usr)
	if err != nil {
		log.Fatal(err) // todo: logger
	}
	svc.mailSvc.SendMessages(
		&core.EmailMessage{
			To:           []mail.Address{{Name: usr.Name, Address: usr.Email}},
			Subject:      "Password Reset",
			TemplateName: "password-reset",
			TemplateData: struct {
				User         User
				PwdResetPath string
			}{usr, fmt.Sprintf("/password-reset/%s/%s", encodeUID(usr), token)},
		},
	)
}

func (svc *service) Delete(ids ...int) error {
	return svc.repo.DeleteUsersByID(ids...)
}
