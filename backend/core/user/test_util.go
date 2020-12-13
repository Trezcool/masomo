package user

import (
	"github.com/trezcool/masomo/core"
)

type serviceMock struct {
	service
}

func NewServiceMock(repo Repository, mailSvc core.EmailService) Service {
	return &serviceMock{
		service: service{
			repo:    repo,
			mailSvc: mailSvc,
		},
	}
}

func (svc *serviceMock) RequestPasswordReset(email string) error {
	usr, err := svc.GetByEmail(email)
	if err != nil {
		return err
	}
	// run synchronously
	svc.sendPasswordResetMail(usr)
	return nil
}
