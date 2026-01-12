package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/example/4-full-maven-repo/model"
	"github.com/example/4-full-maven-repo/repository"
)



type UserService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

func (s *UserService) CreateUser(username, email, password string) (*model.User, error) {
	if username == "" {
		return nil, errors.New("Username cannot be empty")
	}

	if !isValidEmail(email) {
		return nil, errors.New("Invalid email format")
	}

	user := &model.User{}
	user.SetUsername(username)
	user.SetEmail(email)
	user.SetPassword(password)
	user.SetStatus(model.UserStatusActive)

	return s.userRepository.Save(user)
}

func (s *UserService) FindUserById(id int64) *model.User {
	return s.userRepository.FindById(id)
}

func (s *UserService) FindAllActiveUsers() []*model.User {
	return s.userRepository.FindByStatus(model.UserStatusActive)
}

func (s *UserService) UpdateUserStatus(userId int64, newStatus model.UserStatus) (*model.User, error) {
	user := s.userRepository.FindById(userId)
	if user == nil {
		return nil, errors.New("User not found: " + string(rune(userId)))
	}

	user.SetStatus(newStatus)
	return s.userRepository.Save(user)
}

func (s *UserService) DeleteUser(userId int64) bool {
	user := s.userRepository.FindById(userId)
	if user == nil {
		return false
	}

	user.SetStatus(model.UserStatusInactive)
	s.userRepository.Save(user)
	return true
}

func (s *UserService) ValidateUserCredentials(email, password string) bool {
	user := s.userRepository.FindByEmail(email)
	if user == nil {
		return false
	}

	if !user.IsActive() {
		return false
	}

	if user.GetPassword() != password {
		return false
	}

	return true
}

func isValidEmail(email string) bool {
	// This is a placeholder; in real code, you'd use a proper email validation
	return len(email) > 0 && len(email) <= 254
}

// CreateUser creates a new user with the given username, email, and password.
// It validates the input parameters and returns the saved user or an error.

// FindUserByID returns the user with the given ID if it exists.
func (s *UserService) FindUserByID(id int64) (*model.User, error) {
	return userRepository.findById(id)
}

// FindAllActiveUsers returns a list of all active users.
func FindAllActiveUsers() ([]model.User, error) {
	return userRepository.findByStatus(model.UserUserStatusACTIVE)
}

// UpdateUserStatus updates the status of a user identified by userId.
// It returns the updated user or an error if the user is not found.

// No Java source code was provided to translate.

// DeleteUser deletes a user by setting their status to inactive.
// It returns true if the user was found and updated, false otherwise.

// ValidateUserCredentials validates the user credentials by checking if a user with the given email exists,
// is active, and has the provided password.
