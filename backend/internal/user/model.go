package user

import (
	"context"
	"time"
)

type Role string

const (
	RoleStudent   Role = "student"
	RoleTeacher   Role = "teacher"
	RoleModerator Role = "moderator"
)

func (r Role) Valid() bool {
	return r == RoleStudent || r == RoleTeacher || r == RoleModerator
}

type User struct {
	ID                 string    `json:"id"`
	Login              string    `json:"login"`
	FullName           string    `json:"fullName"`
	Role               Role      `json:"role"`
	Timezone           string    `json:"timezone"`
	Active             bool      `json:"active"`
	MustChangePassword bool      `json:"mustChangePassword"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Credentials struct {
	User
	PasswordHash string
}

type CreateInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
	Role     Role   `json:"role"`
	Timezone string `json:"timezone"`
}

type UpdateInput struct {
	FullName *string `json:"fullName"`
	Timezone *string `json:"timezone"`
	Active   *bool   `json:"active"`
}

type Repository interface {
	FindByLogin(ctx context.Context, login string) (*Credentials, error)
	FindByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]User, error)
	ListTeachers(ctx context.Context) ([]User, error)
	ListStudents(ctx context.Context) ([]User, error)
	CreateUser(ctx context.Context, input CreateRecord) (*User, error)
	UpdateUser(ctx context.Context, id string, input UpdateInput) (*User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string, mustChange bool) error
}

type CreateRecord struct {
	Login              string
	PasswordHash       string
	FullName           string
	Role               Role
	Timezone           string
	MustChangePassword bool
}
