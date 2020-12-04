package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	htmltmpl "html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	texttmpl "text/template"
)

var (
	templates = parseTemplates()
	debug     = true // todo: from settings
)

type (
	tmplCacheEntry map[string]interface{}    // {ext: *Template}
	tmplCache      map[string]tmplCacheEntry // {name: {tmplCacheEntry}}

	Attachment struct {
		Content     *bytes.Buffer
		ContentType string
		Filename    string
	}

	EmailMessage struct {
		To          []mail.Address
		Cc          []mail.Address
		Bcc         []mail.Address
		Subject     string
		BodyStr     string // simple text/plain, non-templated content
		Attachments []Attachment

		// templated contents
		TemplateName string // without ext
		TemplateData interface{}
		TextContent  string
		HTMLContent  string
	}

	ContextData struct {
		FrontendBaseURL string
		Data            interface{}
	}

	// EmailService is any service that can send emails
	EmailService interface {
		// SendMessages sends messages concurrently
		SendMessages(messages ...*EmailMessage)
	}
)

func (m *EmailMessage) getContextData() ContextData {
	return ContextData{
		FrontendBaseURL: "http://localhost:8080", // todo: TBD
		Data:            m.TemplateData,
	}
}

func (m *EmailMessage) getTemplate(ext string) (interface{}, bool) {
	cache, ok := templates[m.TemplateName]
	if !ok {
		return nil, ok
	}
	tmplEntry, ok := cache[ext]
	return tmplEntry, ok
}

func (m *EmailMessage) renderText() error {
	if m.BodyStr != "" {
		m.TextContent = m.BodyStr
		return nil
	} else if m.TemplateName == "" {
		return nil
	}

	tmplEntry, ok := m.getTemplate(".txt")
	if !ok {
		return nil
	}
	tmpl, ok := tmplEntry.(*texttmpl.Template)
	if !ok {
		return nil
	}

	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, m.getContextData()); err != nil {
		return err
	}
	m.TextContent = buff.String()
	return nil
}

func (m *EmailMessage) renderHTML() error {
	if m.TemplateName == "" {
		return nil
	}

	tmplEntry, ok := m.getTemplate(".gohtml")
	if !ok {
		return nil
	}
	tmpl, ok := tmplEntry.(*htmltmpl.Template)
	if !ok {
		return nil
	}

	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, m.getContextData()); err != nil {
		return err
	}
	m.HTMLContent = buff.String()
	return nil
}

func (m *EmailMessage) Render() error {
	if err := m.renderText(); err != nil {
		return err
	}
	return m.renderHTML()
}

func (m *EmailMessage) Attach(r io.Reader, filename string, ct ...string) error {
	at := Attachment{Filename: filename}

	// read content
	var content []byte
	if _, err := r.Read(content); err != nil {
		return err
	}
	// base64 encode content
	encoder := base64.NewEncoder(base64.StdEncoding, at.Content)
	if _, err := encoder.Write(content); err != nil {
		return err
	}
	encoder.Close()

	if len(ct) > 0 {
		at.ContentType = ct[0]
	} else {
		at.ContentType = http.DetectContentType(content)
	}
	m.Attachments = append(m.Attachments, at)
	return nil
}

func (m *EmailMessage) AttachFile(path string, contentType ...string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return m.Attach(f, filepath.Base(path), contentType...)
}

func (m *EmailMessage) HasRecipients() bool {
	return len(m.To) > 0
}

func (m *EmailMessage) HasContent() bool {
	return (m.TextContent != "") || (m.HTMLContent != "")
}

func (m *EmailMessage) HasAttachments() bool {
	return len(m.Attachments) > 0
}

func parseTemplates() tmplCache {
	cache := make(tmplCache)

	wd := Getwd()
	rp := filepath.Join(wd, "assets", "templates", "email")
	fps, err := filepath.Glob(filepath.Join(rp, "*"))
	if err != nil {
		log.Fatal(fmt.Errorf("core.parseTemplates: %v", err))
	}

	for _, fp := range fps {
		fname := filepath.Base(fp)
		ext := filepath.Ext(fname)
		if strings.HasPrefix(fname, "_") || !(ext == ".txt" || ext == ".gohtml") {
			continue
		}
		name := fname[:strings.LastIndex(fname, ".")]
		entry, ok := cache[name]
		if !ok {
			cache[name] = make(tmplCacheEntry)
			entry = cache[name]
		}
		if ext == ".txt" {
			tmpl, err := texttmpl.ParseFiles(filepath.Join(rp, "_base.txt"), fp)
			if err != nil {
				log.Fatal(fmt.Errorf("core.parseTemplates: %v", err))
			}
			if debug {
				tmpl = tmpl.Option("missingkey=error")
			}
			entry[ext] = tmpl
		} else {
			tmpl, err := htmltmpl.ParseFiles(filepath.Join(rp, "_base.gohtml"), fp)
			if err != nil {
				log.Fatal(fmt.Errorf("core.parseTemplates: %v", err))
			}
			if debug {
				tmpl = tmpl.Option("missingkey=error")
			}
			entry[ext] = tmpl
		}
	}
	return cache
}
