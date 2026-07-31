package smtp

import "testing"

func TestEmailSend(t *testing.T) {
	t.Skipf("Skip TestEmailSend")
	config := &Config{
		Host:     "smtp.mail.me.com",
		Port:     587,
		User:     "support@alpha.dev",
		Pass:     "password",
		From:     "support@alpha.dev",
		SSL:      true,
		SiteName: "",
	}
	address := []string{"tension@sparkdance.dev"}
	subject := "test"
	body := "test"
	email := NewClient(config)
	err := email.Send(address, subject, body)
	if err != nil {
		t.Errorf("send email error: %v", err)
	}
}
