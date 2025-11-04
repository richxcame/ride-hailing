package notifications

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

// EmailClient handles email operations
type EmailClient struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	fromName     string
}

// NewEmailClient creates a new email client
func NewEmailClient(smtpHost, smtpPort, username, password, fromEmail, fromName string) *EmailClient {
	return &EmailClient{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: username,
		smtpPassword: password,
		fromEmail:    fromEmail,
		fromName:     fromName,
	}
}

// EmailData represents data for email template
type EmailData struct {
	RecipientName string
	Subject       string
	Body          string
	Data          map[string]interface{}
}

// SendEmail sends a plain text email
func (e *EmailClient) SendEmail(to, subject, body string) error {
	from := fmt.Sprintf("%s <%s>", e.fromName, e.fromEmail)

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, body))

	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)

	err := smtp.SendMail(addr, auth, e.fromEmail, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendHTMLEmail sends an HTML email
func (e *EmailClient) SendHTMLEmail(to, subject, htmlBody string) error {
	from := fmt.Sprintf("%s <%s>", e.fromName, e.fromEmail)

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, htmlBody))

	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)

	err := smtp.SendMail(addr, auth, e.fromEmail, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send HTML email: %w", err)
	}

	return nil
}

// Email templates
const (
	welcomeEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .footer { padding: 20px; text-align: center; font-size: 12px; color: #666; }
        .button { background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to RideHailing!</h1>
        </div>
        <div class="content">
            <p>Hi {{.RecipientName}},</p>
            <p>Welcome to RideHailing! We're excited to have you on board.</p>
            <p>{{.Body}}</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 RideHailing. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

	rideConfirmationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #2196F3; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .ride-details { background-color: white; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .detail-row { display: flex; justify-content: space-between; padding: 5px 0; }
        .footer { padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Ride Confirmation</h1>
        </div>
        <div class="content">
            <p>Hi {{.RecipientName}},</p>
            <p>Your ride has been confirmed!</p>
            <div class="ride-details">
                <h3>Ride Details</h3>
                {{range $key, $value := .Data}}
                <div class="detail-row">
                    <strong>{{$key}}:</strong>
                    <span>{{$value}}</span>
                </div>
                {{end}}
            </div>
        </div>
        <div class="footer">
            <p>&copy; 2024 RideHailing. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

	receiptEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .receipt { background-color: white; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .total { font-size: 20px; font-weight: bold; color: #4CAF50; text-align: right; padding-top: 10px; border-top: 2px solid #ddd; }
        .footer { padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Ride Receipt</h1>
        </div>
        <div class="content">
            <p>Hi {{.RecipientName}},</p>
            <p>Thank you for riding with us! Here's your receipt:</p>
            <div class="receipt">
                {{range $key, $value := .Data}}
                <div style="display: flex; justify-content: space-between; padding: 5px 0;">
                    <span>{{$key}}</span>
                    <span>{{$value}}</span>
                </div>
                {{end}}
            </div>
        </div>
        <div class="footer">
            <p>&copy; 2024 RideHailing. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`
)

// SendWelcomeEmail sends a welcome email to new users
func (e *EmailClient) SendWelcomeEmail(to, name string) error {
	tmpl, err := template.New("welcome").Parse(welcomeEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := EmailData{
		RecipientName: name,
		Body:          "Start requesting rides or sign up as a driver to start earning!",
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return e.SendHTMLEmail(to, "Welcome to RideHailing!", body.String())
}

// SendRideConfirmationEmail sends ride confirmation email
func (e *EmailClient) SendRideConfirmationEmail(to, name string, rideDetails map[string]interface{}) error {
	tmpl, err := template.New("confirmation").Parse(rideConfirmationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := EmailData{
		RecipientName: name,
		Data:          rideDetails,
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return e.SendHTMLEmail(to, "Your Ride is Confirmed!", body.String())
}

// SendReceiptEmail sends ride receipt email
func (e *EmailClient) SendReceiptEmail(to, name string, receiptDetails map[string]interface{}) error {
	tmpl, err := template.New("receipt").Parse(receiptEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := EmailData{
		RecipientName: name,
		Data:          receiptDetails,
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return e.SendHTMLEmail(to, "Your Ride Receipt", body.String())
}
