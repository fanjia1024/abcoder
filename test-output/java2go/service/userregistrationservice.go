package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/example/4-full-maven-repo/model"
	"github.com/example/4-full-maven-repo/utils"
)



type UserRegistrationService struct {
	userService  service.UserService
	emailService service.EmailService
}

func NewUserRegistrationService(userService service.UserService, emailService service.EmailService) *UserRegistrationService {
	return &UserRegistrationService{
		userService:  userService,
		emailService: emailService,
	}
}

func (urs *UserRegistrationService) RegisterUser(username, email, password string) *model.User {
	if util.IsEmpty(username) {
		panic("Username is required")
	}

	if !util.IsValidEmail(email) {
		panic("Invalid email format")
	}

	if util.IsEmpty(password) {
		panic("Password is required")
	}

	user := urs.userService.CreateUser(username, email, password)
	urs.emailService.SendWelcomeEmail(user)
	return user
}

func (urs *UserRegistrationService) InitiatePasswordReset(email string) bool {
	if !util.IsValidEmail(email) {
		panic("Invalid email format")
	}

	users := urs.userService.FindAllActiveUsers()
	for _, user := range users {
		if email != "" && email == user.GetEmail() {
			resetToken := urs.generateResetToken()
			urs.emailService.SendPasswordResetEmail(user, resetToken)
			return true
		}
	}
	return false
}

func (urs *UserRegistrationService) generateResetToken() string {
	return fmt.Sprintf("RESET-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
}

// RegisterUser registers a new user with the provided username, email, and password.
// It validates the input parameters, creates the user, and sends a welcome email.
// Returns the created user and an error if any validation fails or an operation encounters an issue.

// InitiatePasswordReset initiates a password reset for the user with the given email.
// It returns true if a reset email was sent, false if no active user was found with that email.
// An error is returned if the email format is invalid.
