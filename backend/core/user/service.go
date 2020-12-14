package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"time"

	"github.com/trezcool/masomo/core"
)

var (
	// errors
	ErrNotFound   = errors.New("user not found")
	ErrUserExists = errors.New("a user with this username or email already exists")
)

type (
	// a sql.Tx is optionally passed to methods as core.DBExecutor for Transaction control only
	Repository interface {
		CheckUsernameUniqueness(ctx context.Context, username, email string, excludedUsers []User, exec ...core.DBExecutor) error
		CreateUser(ctx context.Context, user User, exec ...core.DBExecutor) (User, error)
		// QueryUsers returns all Users or filters them by applying AND operation on available QueryFilter fields.
		// QueryFilter.Search does a case-insensitive match on one of User.Name, User.Username or User.Email.
		QueryUsers(ctx context.Context, filter *QueryFilter, ordering []core.DBOrdering, exec ...core.DBExecutor) ([]User, error)
		GetUser(ctx context.Context, filter GetFilter, exec ...core.DBExecutor) (User, error)
		UpdateUser(ctx context.Context, user User, exec ...core.DBExecutor) (User, error)
		DeleteUsersByID(ctx context.Context, ids []string, exec ...core.DBExecutor) (int, error)
	}

	Service interface {
		CheckUniqueness(uname, email string, exclUsers ...User) error
		Create(nu NewUser) (User, error)
		Query(filter *QueryFilter, ordering []core.DBOrdering) ([]User, error)
		GetByID(id string) (User, error)
		GetByUsername(uname string) (User, error)
		GetByEmail(email string) (User, error)
		GetByUsernameOrEmail(uname string) (User, error)
		Update(id string, uu UpdateUser) (User, error)
		SetLastLogin(usr User) (User, error)
		RequestPasswordReset(email string) error
		ResetPassword(rp ResetUserPassword) error
		Delete(ids ...string) error
	}

	service struct {
		db       core.DB
		repo     Repository
		mailSvc  core.EmailService
		ordering []core.DBOrdering // default
		//log *log.Logger
	}
)

var _ Service = (*service)(nil)

func NewService(db core.DB, repo Repository, mailSvc core.EmailService) Service {
	return &service{
		db:       db,
		repo:     repo,
		mailSvc:  mailSvc,
		ordering: []core.DBOrdering{{Field: "created_at"}},
	}
}

func (svc *service) CheckUniqueness(uname, email string, exclUsers ...User) error {
	if err := svc.repo.CheckUsernameUniqueness(context.Background(), uname, email, exclUsers); err != nil {
		if err == ErrUserExists {
			return core.NewValidationError(err)
		}
		return err
	}
	return nil
}

func (svc *service) Create(nu NewUser) (User, error) {
	usr := User{
		Name:     nu.Name,
		Username: nu.Username,
		Email:    nu.Email,
		Roles:    nu.Roles,
	}
	usr.SetActive(true)
	if err := usr.SetPassword(nu.Password); err != nil {
		return User{}, err
	}
	return svc.repo.CreateUser(context.Background(), usr)
}

func (svc *service) Query(filter *QueryFilter, ordering []core.DBOrdering) ([]User, error) {
	if len(ordering) == 0 {
		ordering = svc.ordering
	}
	return svc.repo.QueryUsers(context.Background(), filter, ordering)
}

func (svc *service) GetByID(id string) (User, error) {
	return svc.repo.GetUser(context.Background(), GetFilter{ID: id})
}

func (svc *service) GetByUsername(uname string) (User, error) {
	return svc.repo.GetUser(context.Background(), GetFilter{Username: core.CleanString(uname, true /* lower */)})
}

func (svc *service) GetByEmail(email string) (User, error) {
	return svc.repo.GetUser(context.Background(), GetFilter{Email: core.CleanString(email, true /* lower */)})
}

func (svc *service) GetByUsernameOrEmail(uname string) (User, error) {
	return svc.repo.GetUser(context.Background(), GetFilter{UsernameOrEmail: core.CleanString(uname, true /* lower */)})
}

func (svc *service) Update(id string, uu UpdateUser) (User, error) {
	usr := User{
		ID:       id,
		Name:     uu.Name,
		Username: uu.Username,
		Email:    uu.Email,
		IsActive: uu.IsActive,
		Roles:    uu.Roles,
	}
	if uu.Password != "" {
		if err := usr.SetPassword(uu.Password); err != nil {
			return User{}, err
		}
	}
	return svc.repo.UpdateUser(context.Background(), usr)
}

func (svc *service) SetLastLogin(usr User) (User, error) {
	usr.LastLogin = time.Now().UTC()
	return svc.repo.UpdateUser(context.Background(), usr)
}

func (svc *service) RequestPasswordReset(email string) error {
	usr, err := svc.GetByEmail(email)
	if err != nil {
		return err
	}
	// do not wait for it; avoid giving clues to attackers
	go svc.sendPasswordResetMail(usr)
	return nil
}

func (svc *service) sendPasswordResetMail(usr User) {
	token, err := MakeToken(usr)
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
			}{usr, fmt.Sprintf("/password-reset/%s/%s", EncodeUID(usr), token)},
		},
	)
}

func (svc *service) ResetPassword(rp ResetUserPassword) error {
	uid, err := decodeUID(rp.UID)
	if err != nil {
		return core.NewValidationError(err, core.FieldError{Field: "uid", Error: "invalid value"})
	}
	usr, err := svc.GetByID(uid)
	if err != nil {
		if err == ErrNotFound {
			return core.NewValidationError(err, core.FieldError{Field: "uid", Error: "invalid value"})
		}
		return err
	}
	if err := verifyToken(usr, rp.Token); err != nil {
		switch err {
		case errInvalidToken, errTokenExpired:
			return core.NewValidationError(err, core.FieldError{Field: "token", Error: "invalid value"})
		default:
			return err
		}
	}

	if err := usr.SetPassword(rp.Password); err != nil {
		return err
	}
	if _, err := svc.repo.UpdateUser(context.Background(), usr); err != nil {
		return err
	}
	return nil
}

func (svc *service) Delete(ids ...string) error {
	if _, err := svc.repo.DeleteUsersByID(context.Background(), ids); err != nil {
		return err
	}
	return nil
}
