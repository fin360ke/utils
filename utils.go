package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"
	"unicode"
)

type EmailConfig struct {
	SMTPServer     string
	SMTPPort       int
	SenderEmail    string
	SenderPassword string
}

func SendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func FormatMobileNumber(number string) string {
	// Trim spaces
	number = strings.TrimSpace(number)

	// Remove any non-digit characters
	numberDigits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, number)

	// Check if the number is valid (at least 9 digits, not more than 12)
	if len(numberDigits) < 9 || len(numberDigits) > 12 {
		return ""
	}

	// Get the last 9 digits
	last9Digits := numberDigits[len(numberDigits)-9:]

	// Format the number
	formattedNumber := "+254" + last9Digits

	// Validate the final format
	if !strings.HasPrefix(formattedNumber, "+2547") && !strings.HasPrefix(formattedNumber, "+2541") {
		return ""
	}

	return formattedNumber
}

func ConvertToISOFormat(dateStr string) string {
	// Try parsing the simplified format first
	t, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err == nil {
		return t.Format(time.RFC3339)
	}
	// If parsing fails, return the original string (assuming it's already in ISO format)
	return dateStr
}

func SendAlert(subject string, message string, emailAddresses []string, phoneNumbers []string) (error error, response string) {
	emailConfig := EmailConfig{
		SMTPServer:     "smtppro.zoho.com",
		SMTPPort:       465, //465 or 587 for TLS
		SenderEmail:    "service@checkitinvestments.com",
		SenderPassword: "J9dGC4NCZdJg", // You may want to use an app password for better security
	}

	currentTime := time.Now().Format("2006-01-02 15:04:05") // Get the current time in a formatted string
	body := message + fmt.Sprintf(". This request was generated at %v", currentTime)

	err := SendEmail(emailConfig, emailAddresses, subject, body)
	if err != nil {
		log.Println("Error sending email:", err)
		return err, "Error sending email:"
	} else {
		log.Println("Email sent successfully.")
	}

	return nil, "Submission successful"
}

func SendEmail(config EmailConfig, toEmails []string, subject, body string) error {
	from := config.SenderEmail
	password := config.SenderPassword

	msg := []byte("To: " + strings.Join(toEmails, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", from, password, config.SMTPServer)

	// Set up the TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         config.SMTPServer,
	}

	// Dial the SMTP server using TLS
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", config.SMTPServer, config.SMTPPort), tlsConfig)
	if err != nil {
		log.Printf("Error sending email: Failed to establish SSL connection: %v", err)
		return err
	}
	defer conn.Close()

	// Create an SMTP client
	client, err := smtp.NewClient(conn, config.SMTPServer)
	if err != nil {
		log.Printf("Error sending email: Failed to create SMTP client: %v", err)
		return err
	}
	defer client.Close()

	// Authenticate with the SMTP server
	if err = client.Auth(auth); err != nil {
		log.Printf("Error sending email: Failed to authenticate: %v", err)
		return err
	}

	// Set the sender and recipients
	if err = client.Mail(from); err != nil {
		log.Printf("Error sending email: Failed to set sender: %v", err)
		return err
	}
	for _, toEmail := range toEmails {
		log.Printf("Attempting to send to %v", toEmail)
		if err = client.Rcpt(toEmail); err != nil {
			log.Printf("Error sending email: Failed to set recipient: %v", err)
			return err
		}
	}

	// Send the email data
	w, err := client.Data()
	if err != nil {
		log.Printf("Error sending email: Failed to start data transfer: %v", err)
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		log.Printf("Error sending email: Failed to write message: %v", err)
		return err
	}
	err = w.Close()
	if err != nil {
		log.Printf("Error sending email: Failed to close data transfer: %v", err)
		return err
	}

	return nil
}
