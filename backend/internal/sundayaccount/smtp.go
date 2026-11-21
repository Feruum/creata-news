package sundayaccount

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

type SMTPMailer struct{ URL, From string }

func (m *SMTPMailer) SendCode(ctx context.Context, to, code string) error {
	u, err := url.Parse(m.URL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "smtp" && u.Scheme != "smtps") {
		return errors.New("invalid SMTP configuration")
	}
	from, err := mail.ParseAddress(m.From)
	if err != nil || strings.ContainsAny(m.From, "\r\n") {
		return errors.New("invalid SMTP sender")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "smtps" {
			port = "465"
		} else {
			port = "587"
		}
	}
	address := net.JoinHostPort(u.Hostname(), port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return errors.New("SMTP connection failed")
	}
	defer conn.Close()
	deadline := time.Now().Add(20 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	tlsConfig := &tls.Config{ServerName: u.Hostname(), MinVersion: tls.VersionTLS12}
	if u.Scheme == "smtps" {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return errors.New("SMTP TLS failed")
		}
		conn = secure
	}
	c, err := smtp.NewClient(conn, u.Hostname())
	if err != nil {
		return errors.New("SMTP greeting failed")
	}
	defer c.Close()
	if u.Scheme == "smtp" {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("SMTP requires STARTTLS")
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			return errors.New("SMTP TLS failed")
		}
	}
	if u.User != nil {
		password, _ := u.User.Password()
		if err := c.Auth(smtp.PlainAuth("", u.User.Username(), password, u.Hostname())); err != nil {
			return errors.New("SMTP authentication failed")
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return errors.New("SMTP sender rejected")
	}
	if err := c.Rcpt(to); err != nil {
		return errors.New("SMTP recipient rejected")
	}
	writer, err := c.Data()
	if err != nil {
		return errors.New("SMTP message rejected")
	}
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: Your Sunday Open sign-in code\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour Sunday Open sign-in code is %s.\r\n\r\nIt expires in 10 minutes. If you did not request this code, ignore this email.\r\n", from.String(), to, code)
	if err != nil {
		return errors.New("SMTP message write failed")
	}
	if err := writer.Close(); err != nil {
		return errors.New("SMTP delivery failed")
	}
	return c.Quit()
}
