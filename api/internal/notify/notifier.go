package notify

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Notifier interface {
	SendEmail(to, subject, body string) error
	SendWhatsApp(to, message string) error
}

type SMTPNotifier struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTPNotifier(host string, port int, user, password, from string) *SMTPNotifier {
	return &SMTPNotifier{host: host, port: port, user: user, password: password, from: from}
}

func (n *SMTPNotifier) SendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", n.host, n.port)
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", n.from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if n.user != "" {
		auth = smtp.PlainAuth("", n.user, n.password, n.host)
	}

	return smtp.SendMail(addr, auth, n.from, []string{to}, []byte(msg))
}

func (n *SMTPNotifier) SendWhatsApp(_ string, _ string) error {
	return fmt.Errorf("whatsapp não implementado")
}

type FakeNotifier struct {
	Emails []string
}

func (f *FakeNotifier) SendEmail(to, subject, body string) error {
	f.Emails = append(f.Emails, fmt.Sprintf("%s|%s|%s", to, subject, body))
	return nil
}

func (f *FakeNotifier) SendWhatsApp(to, message string) error {
	log.Printf("whatsapp stub: to=%s msg=%s", to, message)
	return fmt.Errorf("whatsapp não implementado")
}
