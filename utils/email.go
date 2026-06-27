package utils

import (
	"bytes"
	_ "embed"
	"html/template"

	"github.com/resend/resend-go/v3"
)

var resendClient *resend.Client

func InitResend() {
	resendClient = resend.NewClient(Cfg.ResendAPIKey)
}

//go:embed emails/waitlist.html
var waitlistHTML string

func SendWaitlistEmail(email string) error {

	tmpl, err := template.New("waitlist").Parse(waitlistHTML)
	if err != nil {
		return err
	}

	var body bytes.Buffer

	err = tmpl.Execute(&body, map[string]string{})
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
