package repository

import (
	"strings"
	"sync"
	"time"

	"github.com/example/4-full-maven-repo/model"
)



type InMemoryUserRepository struct {
	users        map[int64]*model.User
	usersByEmail map[string]*model.User
	idGenerator  int64
	mutex        sync.RWMutex
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:        make(map[int64]*model.User),
		usersByEmail: make(map[string]*model.User),
		idGenerator:  1,
	}
}

func (r *InMemoryUserRepository) Save(user *model.User) *model.User {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if user.Id == nil {
		id := r.idGenerator
		r.idGenerator++
		user.Id = &id
	}

	r.users[*user.Id] = user
	if user.Email != nil {
		email := *user.Email
		r.usersByEmail[email] = user
	}

	return user
}

func (r *InMemoryUserRepository) FindById(id int64) *model.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if user, exists := r.users[id]; exists {
		return user
	}
	return nil
}

func (r *InMemoryUserRepository) FindByEmail(email string) *model.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if user, exists := r.usersByEmail[email]; exists {
		return user
	}
	return nil
}

func (r *InMemoryUserRepository) FindByStatus(status model.UserStatus) []*model.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*model.User
	for _, user := range r.users {
		if user.Status == status {
			result = append(result, user)
		}
	}
	return result
}

func (r *InMemoryUserRepository) FindAll() []*model.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*model.User, 0, len(r.users))
	for _, user := range r.users {
		result = append(result, user)
	}
	return result
}

func (r *InMemoryUserRepository) DeleteById(id int64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[id]
	if !exists {
		return
	}

	delete(r.users, id)
	if user.Email != nil {
		delete(r.usersByEmail, *user.Email)
	}
}

func (r *InMemoryUserRepository) Count() int64 {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return int64(len(r.users))
}

func (r *InMemoryUserRepository) ExistsById(id int64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.users[id]
	return exists
}

func (r *InMemoryUserRepository) ExistsByEmail(email string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.usersByEmail[email]
	return exists
}

// Save saves the given user. If the user has no ID, a new ID is assigned.
// Returns the saved user and an error if any occurred.

// FindByID returns the user with the given ID if it exists, otherwise returns nil.
func (r *InMemoryUserRepository) FindByID(id int64) (*model.User, error) {
	user, exists := r.users[id]
	if !exists {
		return nil, nil
	}
	return &user, nil
}

// FindByEmail returns the user with the given email, or nil if not found.

// FindByStatus returns a list of users with the specified status.

// FindAll returns a slice of all users in the repository.

// DeleteByID removes a user by their ID and also removes them from the email index if their email exists.
func (r *InMemoryUserRepository) DeleteByID(id int64) error {
	user, exists := r.users[id]
	if !exists {
		return nil
	}
	delete(r.users, id)
	if user != nil && user.GetEmail() != "" {
		emailKey := strings.ToLower(user.GetEmail())
		delete(r.usersByEmail, emailKey)
	}
	return nil
}

// Count returns the number of users in the repository.

func (r *InMemoryUserRepository) ExistsByID(id int64) bool {
	_, exists := r.users[id]
	return exists
}

// ExistsByEmail checks if a user exists with the given email address (case-insensitive).
