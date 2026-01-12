package service

import (
	"fmt"
	"strings"
)



type EmailService struct{}

func (e *EmailService) SendWelcomeEmail(user *model.User) {
	if user == nil || !isValidEmail(user.Email) {
		panic("Invalid user or email")
	}

	subject := "Welcome to our platform, " + capitalize(user.Username)
	body := fmt.Sprintf(
		"Dear %s,\n\nWelcome to our platform! Your account has been successfully created.\n\nBest regards,\nThe Team",
		capitalize(user.Username),
	)

	// 模拟发送邮件
	fmt.Println("Sending email to:", user.Email)
	fmt.Println("Subject:", subject)
	fmt.Println("Body:", body)
}

func (e *EmailService) SendPasswordResetEmail(user *model.User, resetToken string) {
	if user == nil || !isValidEmail(user.Email) {
		panic("Invalid user or email")
	}

	if isEmpty(resetToken) {
		panic("Reset token cannot be empty")
	}

	subject := "Password Reset Request"
	body := fmt.Sprintf(
		"Dear %s,\n\nYou have requested a password reset. Please use the following token: %s\n\nThis token will expire in 1 hour.\n\nBest regards,\nThe Team",
		capitalize(user.Username),
		resetToken,
	)

	// 模拟发送邮件
	fmt.Println("Sending password reset email to:", user.Email)
	fmt.Println("Subject:", subject)
	fmt.Println("Body:", body)
}

func isValidEmail(email string) bool {
	return len(email) > 0 && strings.Contains(email, "@")
}

func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// SendWelcomeEmail sends a welcome email to the given user.
// It returns an error if the user is nil or has an invalid email address.
func SendWelcomeEmail(user *model.User) error {
	if user == nil || !StringUtils_isValidEmail(user.GetEmail()) {
		return fmt.Errorf("invalid user or email")
	}

	username := StringUtils_capitalize(user.GetUsername())
	subject := "Welcome to our platform, " + username
	body := fmt.Sprintf(
		"Dear %s,\n\nWelcome to our platform! Your account has been successfully created.\n\nBest regards,\nThe Team",
		username,
	)

	// Simulate sending email
	fmt.Printf("Sending email to: %s\n", user.GetEmail())
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Body: %s\n", body)

	return nil
}

// SendPasswordResetEmail sends a password reset email to the given user with the provided reset token.
// It returns an error if the user is nil, the email is invalid, or the reset token is empty.
func SendPasswordResetEmail(user *model.User, resetToken string) error {
	if user == nil || !StringUtils_isValidEmail(user.GetEmail()) {
		return fmt.Errorf("invalid user or email")
	}

	if StringUtils_isEmpty(resetToken) {
		return fmt.Errorf("reset token cannot be empty")
	}

	subject := "Password Reset Request"
	body := fmt.Sprintf(
		"Dear %s,\n\nYou have requested a password reset. Please use the following token: %s\n\nThis token will expire in 1 hour.\n\nBest regards,\nThe Team",
		StringUtils_capitalize(user.GetUsername()),
		resetToken,
	)

	// 模拟发送邮件
	fmt.Printf("Sending password reset email to: %s\n", user.GetEmail())
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Body: %s\n", body)

	return nil
}
