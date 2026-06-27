package utils

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/url"

	"github.com/resend/resend-go/v3"
)

var resendClient *resend.Client

func InitResend() {
	resendClient = resend.NewClient(Cfg.ResendAPIKey)
}

//go:embed emails/waitlist.html
var waitlistHTML string

func SendWaitlistEmail(email, token string) error {
	confirmURL := "https://devolib.com/waitlist/confirm?token=" + url.QueryEscape(token)

	tmpl, err := template.New("waitlist").Parse(waitlistHTML)
	if err != nil {
		return err
	}

	var body bytes.Buffer

	err = tmpl.Execute(&body, map[string]string{
		"ConfirmURL": confirmURL,
	})
	if err != nil {
		return err
	}

	params := &resend.SendEmailRequest{
		From:    "team@devolib.com",
		To:      []string{email},
		Subject: "Thanks for joining us!",
		Html:    body.String(),
	}

	_, err = resendClient.Emails.Send(params)
	return err
}
