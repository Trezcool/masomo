package emailsvc

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/trezcool/masomo/core"
)

var (
	SentMessages = make([]core.EmailMessage, 0)
	mu           sync.Mutex
)

type consoleService struct {
	defaultFromEmail mail.Address
	subjPrefix       string
	disableOutput    bool
}

var _ core.EmailService = (*consoleService)(nil)

func NewConsoleService() core.EmailService {
	return &consoleService{
		defaultFromEmail: core.Conf.DefaultFromEmail,
		subjPrefix:       "[" + core.Conf.AppName + "] ",
	}
}

func (svc consoleService) SendMessages(messages ...*core.EmailMessage) {
	for _, msg := range messages {
		go svc.sendMessage(msg)
	}
}

func (svc consoleService) sendMessage(msg *core.EmailMessage) {
	err := msg.Render()
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "rendering email"))
	}
	if msg.HasRecipients() && (msg.HasContent() || msg.HasAttachments()) {
		svc.send(*msg)
		mu.Lock()
		SentMessages = append(SentMessages, *msg)
		mu.Unlock()
	}
}

func (svc consoleService) send(msg core.EmailMessage) {
	body := new(strings.Builder)

	// Write mail header
	_, _ = fmt.Fprintf(body, "From: %s\r\n", svc.defaultFromEmail.String())
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
			log.Fatalf("%+v", errors.Wrap(err, "creating multipart/alternative part"))
		}
	}

	w, err := altW.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}})
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "creating text/plain part"))
	}
	_, _ = fmt.Fprintf(w, "%s\r\n", msg.TextContent)

	if msg.TemplateName != "" {
		w, err = altW.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}})
		if err != nil {
			log.Fatalf("%+v", errors.Wrap(err, "creating text/html part"))
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
				log.Fatalf("%+v", errors.Wrap(err, "creating "+at.ContentType+" part"))
			}
			_, _ = fmt.Fprintf(w, "%s\r\n", at.Content.String())
		}
	}

	if !svc.disableOutput {
		log.Println(body.String())
	}
}

func (svc consoleService) joinAddresses(addrs []mail.Address) string {
	toJoin := make([]string, 0, len(addrs))
	for _, a := range addrs {
		toJoin = append(toJoin, a.String())
	}
	return strings.Join(toJoin, ", ")
}

type consoleServiceMock struct {
	consoleService
}

func NewConsoleServiceMock() core.EmailService {
	return &consoleServiceMock{
		consoleService: consoleService{
			defaultFromEmail: core.Conf.DefaultFromEmail,
			subjPrefix:       "[" + core.Conf.AppName + "] ",
			disableOutput:    true,
		},
	}
}

func (svc *consoleServiceMock) SendMessages(messages ...*core.EmailMessage) {
	for _, msg := range messages {
		// run synchronously
		svc.sendMessage(msg)
	}
}
