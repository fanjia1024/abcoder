package model

import (
	"time"
)




type BaseEntity struct {
	ID        *int64    `json:"id,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

func (b *BaseEntity) GetID() *int64 {
	return b.ID
}

func (b *BaseEntity) SetID(id *int64) {
	b.ID = id
}

func (b *BaseEntity) GetCreatedAt() time.Time {
	return b.CreatedAt
}

func (b *BaseEntity) SetCreatedAt(createdAt time.Time) {
	b.CreatedAt = createdAt
}

func (b *BaseEntity) GetUpdatedAt() time.Time {
	return b.UpdatedAt
}

func (b *BaseEntity) SetUpdatedAt(updatedAt time.Time) {
	b.UpdatedAt = updatedAt
}

func (b *BaseEntity) GetCreatedBy() string {
	return b.CreatedBy
}

func (b *BaseEntity) SetCreatedBy(createdBy string) {
	b.CreatedBy = createdBy
}

func (b *BaseEntity) GetUpdatedBy() string {
	return b.UpdatedBy
}

func (b *BaseEntity) SetUpdatedBy(updatedBy string) {
	b.UpdatedBy = updatedBy
}

// GetID returns the ID of the entity.

// GetCreatedAt returns the creation timestamp.

// GetUpdatedAt returns the updated at timestamp.

// GetCreatedBy returns the user who created the entity.

// GetUpdatedBy returns the updatedBy field.
