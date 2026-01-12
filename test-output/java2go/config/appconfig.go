package config

import (
	"context"
	"time"

	"github.com/example/4-full-maven-repo/model"
	"github.com/example/4-full-maven-repo/service"
)



type AppConfig struct{}

func (a *AppConfig) UserRepository() repository.UserRepository {
	return &repository.InMemoryUserRepository{}
}

func (a *AppConfig) UserService(userRepository repository.UserRepository) *service.UserService {
	return service.NewUserService(userRepository)
}

func (a *AppConfig) EmailService() *service.EmailService {
	return &service.EmailService{}
}

func (a *AppConfig) UserRegistrationService(userService *service.UserService, emailService *service.EmailService) *service.UserRegistrationService {
	return service.NewUserRegistrationService(userService, emailService)
}

// repository.UserRepository returns a new in-memory user repository instance.
func UserRepository() repository.UserRepository {
	return &InMemoryUserRepository{}
}

// service.UserService creates and returns a new service.UserService instance.
func UserService(userRepository repository.UserRepository) service.UserService {
	return service.UserService{
		UserRepository: userRepository,
	}
}

// service.EmailService creates and returns a new instance of EmailService.
func EmailService() *EmailService.EmailService {
	return &EmailService.EmailService{}
}

// service.UserRegistrationService creates a new service.UserRegistrationService instance.
func UserRegistrationService(userService service.UserService, emailService service.EmailService) *service.UserRegistrationService {
	return &UserRegistrationService{
		userService:  userService,
		emailService: emailService,
	}
}
