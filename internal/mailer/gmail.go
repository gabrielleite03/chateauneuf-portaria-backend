package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Gmail struct {
	host, username, password, from, bcc string
	port                                int
}

func NewGmail(host string, port int, username, password, from string) *Gmail {
	if strings.TrimSpace(from) == "" {
		from = username
	}
	return &Gmail{host: strings.TrimSpace(host), port: port, username: strings.TrimSpace(username), password: password, from: strings.TrimSpace(from)}
}

func (g *Gmail) SetBCC(bcc string) {
	g.bcc = strings.TrimSpace(bcc)
}

func (g *Gmail) Send(ctx context.Context, to, subject, htmlBody string) error {
	return g.send(ctx, to, subject, htmlBody, "", "", "", nil)
}

func (g *Gmail) SendAttachment(ctx context.Context, to, subject, htmlBody, filename string, attachment []byte) error {
	return g.send(ctx, to, subject, htmlBody, filename, "application/pdf", "", attachment)
}

func (g *Gmail) SendInlineImage(ctx context.Context, to, subject, htmlBody, contentType string, image []byte) error {
	return g.send(ctx, to, subject, htmlBody, "foto-entrega.jpg", contentType, "delivery-photo", image)
}

func (g *Gmail) send(ctx context.Context, to, subject, htmlBody, filename, attachmentType, contentID string, attachment []byte) error {
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
	if g.bcc != "" && !strings.EqualFold(g.bcc, strings.TrimSpace(to)) {
		if err = client.Rcpt(g.bcc); err != nil {
			return fmt.Errorf("definir copia oculta: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("iniciar mensagem: %w", err)
	}
	var message bytes.Buffer
	message.WriteString("From: " + g.from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\n")
	if len(attachment) == 0 {
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n" + htmlBody)
	} else {
		boundary := "chateauneuf-message"
		multipartType := "mixed"
		if contentID != "" {
			multipartType = "related"
		}
		message.WriteString("Content-Type: multipart/" + multipartType + "; boundary=\"" + boundary + "\"\r\n\r\n")
		message.WriteString("--" + boundary + "\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + htmlBody + "\r\n")
		disposition := "attachment"
		cidHeader := ""
		if contentID != "" {
			disposition = "inline"
			cidHeader = "Content-ID: <" + contentID + ">\r\n"
		}
		message.WriteString("--" + boundary + "\r\nContent-Type: " + attachmentType + "\r\nContent-Disposition: " + disposition + "; filename=\"" + filename + "\"\r\n" + cidHeader + "Content-Transfer-Encoding: base64\r\n\r\n")
		encoded := base64.StdEncoding.EncodeToString(attachment)
		for len(encoded) > 76 {
			message.WriteString(encoded[:76] + "\r\n")
			encoded = encoded[76:]
		}
		message.WriteString(encoded + "\r\n--" + boundary + "--\r\n")
	}
	if _, err = w.Write(message.Bytes()); err != nil {
		_ = w.Close()
		return fmt.Errorf("enviar mensagem: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("finalizar mensagem: %w", err)
	}
	return client.Quit()
}
