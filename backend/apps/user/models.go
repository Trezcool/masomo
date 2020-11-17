package user

import (
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/trezcool/masomo/backend/apps/utils"
)

// Roles
const (
	// Admin
	RoleAdmin          = "admin:"
	RoleAdminOwner     = "admin:owner"
	RoleAdminPrincipal = "admin:principal"

	// Teacher
	RoleTeacher = "teacher:"

	// Student
	RoleStudent = "student:"
)

var (
	AdminRoles   = []string{RoleAdmin, RoleAdminOwner, RoleAdminPrincipal}
	TeacherRoles = []string{RoleTeacher}
	StudentRoles = []string{RoleStudent}
	AllRoles     = make([]string, 0, 5)

	rolePriorities = map[string]int{
		// Admins: 30 - 21
		RoleAdminOwner:     30,
		RoleAdminPrincipal: 29,
		RoleAdmin:          21,

		// Teachers: 20 - 11
		RoleTeacher: 11,

		// Students: 10 - 1
		RoleStudent: 1,
	}
)

func init() {
	AllRoles = append(AllRoles, AdminRoles...)
	AllRoles = append(AllRoles, TeacherRoles...)
	AllRoles = append(AllRoles, StudentRoles...)
}

func RolePriority(role string) int {
	return rolePriorities[role]
}

func MaxRolePriority(roles []string) int {
	var max int
	for _, role := range roles {
		if RolePriority(role) > max {
			max = RolePriority(role)
		}
	}
	return max
}

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	IsActive     bool      `json:"is_active"`
	Roles        []string  `json:"roles"`
	PasswordHash []byte    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) SetPassword(pwd string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

func (u *User) CheckPassword(pwd []byte) error {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, pwd)
}

func (u *User) roleStartsWith(prefix string) bool {
	for _, role := range u.Roles {
		if strings.HasPrefix(role, prefix) {
			return true
		}
	}
	return false
}

func (u *User) IsAdmin() bool {
	return u.roleStartsWith(RoleAdmin)
}

func (u *User) IsTeacher() bool {
	return u.roleStartsWith(RoleTeacher)
}

func (u *User) IsStudent() bool {
	return u.roleStartsWith(RoleStudent)
}

// NewUser contains information needed to create a new User.
type NewUser struct {
	Name            string   `json:"name" validate:"required"`
	Username        string   `json:"username" validate:"omitempty,min=6,alphanum_"`
	Email           string   `json:"email" validate:"omitempty,email"`
	Password        string   `json:"password" validate:"required"`
	PasswordConfirm string   `json:"password_confirm" validate:"required,eqfield=Password"`
	Roles           []string `json:"roles" validate:"omitempty,allroles"`
}

func (nu *NewUser) Validate(repo *Repository) error {
	nu.Name = utils.CleanString(nu.Name)
	nu.Username = utils.CleanString(nu.Username, true)
	nu.Email = utils.CleanString(nu.Email, true)

	if err := utils.Validate.Struct(nu); err != nil {
		return err
	}
	return repo.checkUniqueness(nu.Username, nu.Email)
}

// UpdateUser defines what information may be provided to modify an existing User.
type UpdateUser struct {
	Name            string
	Username        string
	Email           string
	IsActive        bool
	Roles           []string
	Password        string
	PasswordConfirm string
}

type QueryFilter struct {
	ID              int
	Name            string
	Username        string
	Email           string
	UsernameOrEmail string
	Roles           []string
	CreatedAt       time.Time
}
