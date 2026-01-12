package model

import (
	"fmt"
	"strings"
)




type User struct {
	BaseEntity
	Username string     `json:"username"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Status   UserStatus `json:"status"`
}

type UserStatus int

const (
	Active UserStatus = iota
	Inactive
	Suspended
)

func (u *User) SetUsername(username string) {
	if username != "" {
		u.Username = strings.TrimSpace(username)
	}
}

func (u *User) GetUsername() string {
	return u.Username
}

func (u *User) SetEmail(email string) {
	if isValidEmail(email) {
		u.Email = strings.ToLower(email)
	}
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) SetPassword(password string) {
	if password != "" {
		u.Password = password
	}
}

func (u *User) GetPassword() string {
	return u.Password
}

func (u *User) SetStatus(status UserStatus) {
	u.Status = status
}

func (u *User) GetStatus() UserStatus {
	return u.Status
}

func (u *User) IsActive() bool {
	return u.Status == Active
}

func (u *User) String() string {
	return "User{" +
		"id=" + u.GetID() +
		", username='" + u.Username + "'" +
		", email='" + u.Email + "'" +
		", status=" + u.statusToString() +
		"}"
}

func (u *User) statusToString() string {
	switch u.Status {
	case Active:
		return "ACTIVE"
	case Inactive:
		return "INACTIVE"
	case Suspended:
		return "SUSPENDED"
	default:
		return "UNKNOWN"
	}
}

// isValidEmail is a placeholder for the actual email validation logic
// that would be equivalent to StringUtils.isValidEmail in Java.
func isValidEmail(email string) bool {
	// This is a minimal placeholder; replace with proper validation as needed.
	return email != "" && strings.Contains(email, "@")
}

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusInactive  UserStatus = "INACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

// GetUsername returns the username.

// GetEmail returns the email address.

// GetPassword returns the password.

// GetStatus returns the status of the user.

// IsActive returns true if the user's status is active.
