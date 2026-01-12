package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)



type UserController struct {
	userService         service.UserService
	registrationService service.UserRegistrationService
}

func NewUserController(userService service.UserService, registrationService service.UserRegistrationService) *UserController {
	return &UserController{
		userService:         userService,
		registrationService: registrationService,
	}
}

func (uc *UserController) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var request UserRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user, err := uc.registrationService.RegisterUser(request.Username, request.Email, request.Password)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (uc *UserController) GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user := uc.userService.FindUserByID(id)
	if user == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (uc *UserController) GetAllActiveUsers(w http.ResponseWriter, r *http.Request) {
	users := uc.userService.FindAllActiveUsers()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (uc *UserController) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	statusStr := r.URL.Query().Get("status")
	status, err := model.ParseUserStatus(statusStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user, err := uc.userService.UpdateUserStatus(id, status)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (uc *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	deleted := uc.userService.DeleteUser(id)
	if !deleted {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (uc *UserController) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	initiated := uc.registrationService.InitiatePasswordReset(request.Email)
	if !initiated {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type UserRegistrationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PasswordResetRequest is already defined in dependencies, so we do not redeclare it.

// RegisterUser registers a new user based on the provided registration request.

// GetUserByID retrieves a user by their ID.
// Returns the user and http.StatusOK if found, or nil and http.StatusNotFound if not found.

// GetAllActiveUsers returns all active users with an HTTP 200 OK status.

// UpdateUserStatus updates the status of a user by ID.

// NotFoundError represents a not found error.
type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}

// DeleteUser deletes a user by ID.

// ResetPassword handles the password reset request.

func (u *UserRegistrationRequest) GetUsername() string {
	return u.Username
}

func (u *UserRegistrationRequest) SetUsername(username string) {
	u.Username = username
}

func (u *UserRegistrationRequest) GetEmail() string {
	return u.Email
}

func (u *UserRegistrationRequest) SetEmail(email string) {
	u.Email = email
}

func (u *UserRegistrationRequest) GetPassword() string {
	return u.Password
}

func (u *UserRegistrationRequest) SetPassword(password string) {
	u.Password = password
}

// GetUsername returns the username.

// GetEmail returns the email address.

// GetEmail returns the email address.
func (p *PasswordResetRequest) GetEmail() string {
	return p.email
}

// GetPassword returns the password.

// SetPassword sets the password for the user registration request.

type PasswordResetRequest struct {
	Email string `json:"email"`
}

func (p *PasswordResetRequest) SetEmail(email string) {
	p.Email = email
}

// No Java source code was provided to translate.

// SetEmail sets the email address.

// SetEmail sets the email address.
