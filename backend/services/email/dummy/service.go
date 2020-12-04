package dummymail

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/trezcool/masomo/backend/core"
)

var SentMessages = make([]core.EmailMessage, 0)

type service struct {
	defaultFromEmail string
	subjPrefix       string
}

var _ core.EmailService = (*service)(nil)

func NewService(appName, defaultFromEmail string) core.EmailService {
	return &service{
		defaultFromEmail: defaultFromEmail,
		subjPrefix:       "[" + appName + "] ",
	}
}

func (svc service) SendMessages(messages ...*core.EmailMessage) {
	for _, msg := range messages {
		err := msg.Render()
		if err != nil {
			log.Fatal(err)
		}
		if msg.HasRecipients() && (msg.HasContent() || msg.HasAttachments()) {
			go svc.send(*msg)
			SentMessages = append(SentMessages, *msg)
		}
	}
}

func (svc service) send(msg core.EmailMessage) {
	body := &strings.Builder{}

	// Write mail header
	_, _ = fmt.Fprintf(body, "From: %s\r\n", svc.defaultFromEmail)
	_, _ = fmt.Fprint(body, "MIME-Version: 1.0\r\n")
	_, _ = fmt.Fprintf(body, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	_, _ = fmt.Fprintf(body, "Subject: %s\r\n", svc.subjPrefix+msg.Subject)
	_, _ = fmt.Fprintf(body, "To: %s\r\n", svc.joinAddresses(msg.To))
	_, _ = fmt.Fprintf(body, "CC: %s\r\n", svc.joinAddresses(msg.Cc))
	_, _ = fmt.Fprintf(body, "BCC: %s\r\n", svc.joinAddresses(msg.Bcc))

	var mixedW *multipart.Writer
	altW := multipart.NewWriter(body)
	defer altW.Close()

	if msg.HasAttachments() {
		mixedW = multipart.NewWriter(body)
		defer mixedW.Close()
		_, _ = fmt.Fprintf(body, "Content-Type: multipart/mixed\r\n")
		_, _ = fmt.Fprintf(body, "Content-Type: boundary=%s\r\n", mixedW.Boundary())
	} else {
		_, _ = fmt.Fprintf(body, "Content-Type: multipart/alternative\r\n")
		_, _ = fmt.Fprintf(body, "Content-Type: boundary=%s\r\n", altW.Boundary())
	}
	_, _ = fmt.Fprint(body, "\r\n")

	if mixedW != nil {
		if _, err := mixedW.CreatePart(textproto.MIMEHeader{"Content-Type": {"multipart/alternative", "boundary=" + altW.Boundary()}}); err != nil {
			log.Fatal(err)
		}
	}

	w, err := altW.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}})
	if err != nil {
		log.Fatal(err)
	}
	_, _ = fmt.Fprintf(w, "%s\r\n", msg.TextContent)

	if msg.TemplateName != "" {
		w, err = altW.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}})
		if err != nil {
			log.Fatal(err)
		}
		_, _ = fmt.Fprintf(w, "%s\r\n", msg.HTMLContent)
	}

	if mixedW != nil {
		for _, at := range msg.Attachments {
			w, err = mixedW.CreatePart(textproto.MIMEHeader{
				"Content-Type":              {at.ContentType},
				"Content-Transfer-Encoding": {"base64"},
				"Content-Disposition":       {"attachment; filename=" + at.Filename}})
			if err != nil {
				log.Fatal(err)
			}
			_, _ = fmt.Fprintf(w, "%s\r\n", at.Content.String())
		}
	}

	log.Println(body.String())
}

func (svc service) joinAddresses(addrs []mail.Address) string {
	toJoin := make([]string, 0, len(addrs))
	for _, a := range addrs {
		toJoin = append(toJoin, a.String())
	}
	return strings.Join(toJoin, ", ")
}
