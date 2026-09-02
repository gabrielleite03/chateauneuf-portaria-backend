package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Gmail struct {
	host, username, password, from string
	port                           int
}

func NewGmail(host string, port int, username, password, from string) *Gmail {
	if strings.TrimSpace(from) == "" {
		from = username
	}
	return &Gmail{host: strings.TrimSpace(host), port: port, username: strings.TrimSpace(username), password: password, from: strings.TrimSpace(from)}
}

func (g *Gmail) Send(ctx context.Context, to, subject, htmlBody string) error {
	if g.host == "" || g.port <= 0 || g.username == "" || g.password == "" || g.from == "" {
		return fmt.Errorf("configuracao SMTP do Gmail incompleta")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", g.host, g.port))
	if err != nil {
		return fmt.Errorf("conectar ao Gmail: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, g.host)
	if err != nil {
		return fmt.Errorf("iniciar SMTP: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: g.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("iniciar TLS SMTP: %w", err)
		}
	}
	if err = client.Auth(smtp.PlainAuth("", g.username, g.password, g.host)); err != nil {
		return fmt.Errorf("autenticar no Gmail: %w", err)
	}
	if err = client.Mail(g.from); err != nil {
		return fmt.Errorf("definir remetente: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("definir destinatario: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("iniciar mensagem: %w", err)
	}
	message := "From: " + g.from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + htmlBody
	if _, err = w.Write([]byte(message)); err != nil {
		_ = w.Close()
		return fmt.Errorf("enviar mensagem: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("finalizar mensagem: %w", err)
	}
	return client.Quit()
}
