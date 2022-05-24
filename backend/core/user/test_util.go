package user

import (
	"github.com/trezcool/masomo/core"
)

type serviceMock struct {
	Service
}

func NewServiceMock(db core.DB, repo Repository, mailSvc core.EmailService, conf *core.Config) *serviceMock {
	return &serviceMock{
		Service: Service{
			db:       db,
			repo:     repo,
			conf:     conf,
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
