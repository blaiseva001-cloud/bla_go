package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/blaiseva001-cloud/backend/internal/config"
)

const otpHTML = `<!doctype html><html><body style="margin:0;background:#040604;padding:32px;font-family:monospace">
<div style="max-width:480px;margin:auto;background:#0c120d;border:1px solid #1a2b1f;padding:32px">
<div style="color:#00ff41;font-size:20px;font-weight:bold">bla.link_</div>
<p style="color:#e9f7ec">your verification code:</p>
<div style="font-size:36px;letter-spacing:12px;color:#00ff41;background:#090d0a;border:1px dashed #2c4c35;padding:16px;text-align:center">%s</div>
<p style="color:#5a8165;font-size:12px">expires in %s minutes. do not share this code.</p>
</div></body></html>`

func SendOTPEmail(cfg *config.Config, to, code string, exp int) error {
	if cfg.SMTPUser == "" {
		return nil
	}
	host := cfg.SMTPHost
	addr := net.JoinHostPort(host, cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, host)
	msg := []byte("From: " + cfg.SMTPFromName + " <" + cfg.SMTPFromEmail + ">\r\n" +
		"To: " + to + "\r\n" +
		"Subject: bla.link verification code\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		fmt.Sprintf(otpHTML, code, fmt.Sprint(exp)))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()
	if ok, _ := client.Extension("STARTTLS"); ok {
		_ = client.StartTLS(&tls.Config{ServerName: host})
	}
	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(cfg.SMTPFromEmail); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}
