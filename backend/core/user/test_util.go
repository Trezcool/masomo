package user

import (
	"github.com/trezcool/masomo/core"
)

type serviceMock struct {
	service
}

func NewServiceMock(db core.DB, repo Repository, mailSvc core.EmailService) *serviceMock {
	return &serviceMock{
		service: service{
			db:       db,
			repo:     repo,
			mailSvc:  mailSvc,
			ordering: []core.DBOrdering{{Field: "created_at"}},
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
