package user

import (
	"time"

	"github.com/trezcool/masomo/backend/core"
)

type serviceMock struct {
	service
}

func NewServiceMock(repo Repository, mailSvc core.EmailService, secret []byte, pwdResetTimeout time.Duration) Service {
	secretKey = secret
	passwordResetTimeoutDelta = pwdResetTimeout
	return &serviceMock{
		service: service{
			repo:    repo,
			mailSvc: mailSvc,
		},
	}
}

func (svc *serviceMock) RequestPasswordReset(email string) error {
	usr, err := svc.repo.GetUserByEmail(email)
	if err != nil {
		return err
	}
	// run synchronously
	svc.sendPasswordResetMail(usr)
	return nil
}
