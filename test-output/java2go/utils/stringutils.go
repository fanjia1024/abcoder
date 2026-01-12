package utils

import (
	"regexp"
	"strings"
)



var emailPattern = regexp.MustCompile(`^[A-Za-z0-9+_.-]+@(.+)$`)

type StringUtils struct{}

func (s *StringUtils) IsEmpty(str string) bool {
	return str == "" || strings.TrimSpace(str) == ""
}

func (s *StringUtils) IsNotEmpty(str string) bool {
	return !s.IsEmpty(str)
}

func (s *StringUtils) Trim(str string) string {
	if str == "" {
		return ""
	}
	return strings.TrimSpace(str)
}

func (s *StringUtils) IsValidEmail(email string) bool {
	return email != "" && emailPattern.MatchString(email)
}

func (s *StringUtils) Capitalize(str string) string {
	if s.IsEmpty(str) {
		return str
	}
	return strings.ToUpper(str[:1]) + strings.ToLower(str[1:])
}

// IsEmpty returns true if the given string is nil or consists solely of whitespace characters.
func IsEmpty(str string) bool {
	return str == "" || strings.TrimSpace(str) == ""
}

// IsNotEmpty returns true if the given string is not empty.
func IsNotEmpty(str string) bool {
	return strings.TrimSpace(str) != ""
}

// Trim returns an empty string if the input is nil, otherwise returns the trimmed string.
func Trim(str string) string {
	if str == "" {
		return ""
	}
	return strings.TrimSpace(str)
}

// isValidEmail checks if the provided email string matches the expected email pattern.
// It returns false if the email is nil or does not match the pattern.
func isValidEmail(email string) bool {
	return email != "" && EMAIL_PATTERN.MatchString(email)
}

// Capitalize returns a copy of the string with the first character converted to uppercase
// and the rest converted to lowercase. If the input string is empty, it returns the empty string.
func Capitalize(str string) string {
	if IsEmpty(str) {
		return str
	}
	return strings.ToUpper(str[:1]) + strings.ToLower(str[1:])
}
