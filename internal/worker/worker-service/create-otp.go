package worker_service

import (
	"fmt"

	"github.com/xenn00/chat-system/config"
	"gopkg.in/gomail.v2"
)

func SendMailTrapOTP(userId, otp string) error {
	host := config.Conf.MAILTRAP.SMTPHost
	port := config.Conf.MAILTRAP.SMTPPort
	username := config.Conf.MAILTRAP.Username
	password := config.Conf.MAILTRAP.Password
	from := config.Conf.MAILTRAP.From
	to := config.Conf.MAILTRAP.TO // testing only, later we use real-email

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Your OTP Code - Don't share it with others")
	m.SetBody("text/plain", fmt.Sprintf("Hello user %s,\n\nYour OTP code is: %s\n\nValid for 5 minutes.", userId, otp))

	d := gomail.NewDialer(host, port, username, password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}
