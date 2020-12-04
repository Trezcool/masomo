package sendgridmail

import (
	"net/http"
	"net/mail"

	"github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/trezcool/masomo/backend/core"
)

var (
	host     = "https://api.sendgrid.com"
	endpoint = "/v3/mail/send"
)

type service struct {
	key        string
	from       *sgmail.Email
	subjPrefix string
	//logger
}

var _ core.EmailService = (*service)(nil)

func NewService(key, appName, fromEmail string) core.EmailService {
	return &service{
		key:        key,
		from:       sgmail.NewEmail(appName, fromEmail),
		subjPrefix: "[" + appName + "] ",
	}
}

func (svc *service) SendMessages(messages ...*core.EmailMessage) {
	for _, msg := range messages {
		msg := msg
		go func() {
			err := msg.Render()
			if err != nil {
				panic(err) // todo: logger
			}
			if msg.HasRecipients() && (msg.HasContent() || msg.HasAttachments()) {
				svc.send(*msg)
			}
		}()
	}
}

func (svc *service) prepare(msg core.EmailMessage) (*sgmail.SGMailV3, error) {
	p := sgmail.NewPersonalization()
	p.Subject = svc.subjPrefix + msg.Subject

	for _, to := range msg.To {
		p.AddTos(svc.getSGEmail(to))
	}
	for _, cc := range msg.Cc {
		p.AddCCs(svc.getSGEmail(cc))
	}
	for _, bcc := range msg.Bcc {
		p.AddBCCs(svc.getSGEmail(bcc))
	}

	m := sgmail.NewV3Mail()
	m.SetFrom(svc.from)
	m.AddPersonalizations(p)

	m.AddContent(
		sgmail.NewContent("text/plain", msg.TextContent),
		sgmail.NewContent("text/html", msg.HTMLContent),
	)

	for _, a := range msg.Attachments {
		m.AddAttachment(svc.getSGAttachment(a))
	}

	return m, nil
}

func (svc service) getSGEmail(addr mail.Address) *sgmail.Email {
	return sgmail.NewEmail(addr.Name, addr.Address)
}

func (svc service) getSGAttachment(at core.Attachment) *sgmail.Attachment {
	return &sgmail.Attachment{
		Content:     at.Content.String(),
		Type:        at.ContentType,
		Filename:    at.Filename,
		Disposition: "attachment",
	}
}

func (svc service) send(msg core.EmailMessage) {
	req := sendgrid.GetRequest(svc.key, endpoint, host)
	req.Method = http.MethodPost
	m, err := svc.prepare(msg)
	if err != nil {
		panic(err) // todo: logger
	}
	req.Body = sgmail.GetRequestBody(m)

	res, err := sendgrid.API(req)
	if err != nil || res.StatusCode >= http.StatusBadRequest { // todo: retries !!
		panic(err)
		// todo webhook to handle failed mails ??
	}
}
