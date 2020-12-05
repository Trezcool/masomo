package user

import (
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/trezcool/masomo/backend/core"
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
	AllRoles     = getAllRoles()

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

	Roles = []Role{
		{Name: "Student", Value: RoleStudent},
		{Name: "Teacher", Value: RoleTeacher},
		{Name: "Admin", Value: RoleAdmin},
		{Name: "Admin Principal", Value: RoleAdminPrincipal},
		{Name: "Admin Owner", Value: RoleAdminOwner},
	}
)

func getAllRoles() []string {
	all := make([]string, 0, 5)
	all = append(all, AdminRoles...)
	all = append(all, TeacherRoles...)
	all = append(all, StudentRoles...)
	return all
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

type Role struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	IsActive     bool      `json:"is_active"`
	Roles        []string  `json:"roles"`
	PasswordHash []byte    `json:"-"`
	CreatedAt    time.Time `json:"created_at"` // UTC
	UpdatedAt    time.Time `json:"updated_at"` // UTC
	LastLogin    time.Time `json:"last_login"` // UTC
}

func (u *User) SetPassword(pwd string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

func (u *User) CheckPassword(pwd string) error {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(pwd))
}

func (u *User) RoleStartsWith(prefix string) bool {
	for _, role := range u.Roles {
		if strings.HasPrefix(role, prefix) {
			return true
		}
	}
	return false
}

func (u *User) IsAdmin() bool {
	return u.RoleStartsWith(RoleAdmin)
}

func (u *User) IsTeacher() bool {
	return u.RoleStartsWith(RoleTeacher)
}

func (u *User) IsStudent() bool {
	return u.RoleStartsWith(RoleStudent)
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

func (nu *NewUser) Validate(svc Service) error {
	nu.Name = core.CleanString(nu.Name)
	nu.Username = core.CleanString(nu.Username, true /* lower */)
	nu.Email = core.CleanString(nu.Email, true /* lower */)

	if err := core.Validate.Struct(nu); err != nil {
		return err
	}
	return svc.CheckUniqueness(nu.Username, nu.Email)
}

// UpdateUser defines what information may be provided to modify an existing User.
type UpdateUser struct {
	Name            string   `json:"name"`
	Username        string   `json:"username" validate:"omitempty,min=6,alphanum_"`
	Email           string   `json:"email" validate:"omitempty,email"`
	IsActive        *bool    `json:"is_active"`
	Roles           []string `json:"roles" validate:"omitempty,allroles"`
	Password        string   `json:"password" validate:"omitempty"`
	PasswordConfirm string   `json:"password_confirm" validate:"required_with=Password,eqfield=Password"`
}

func (uu *UpdateUser) Validate(origUsr User, svc Service) error {
	name := core.CleanString(uu.Name)
	if name != "" {
		uu.Name = name
	} else {
		uu.Name = origUsr.Name
	}

	uname := core.CleanString(uu.Username, true /* lower */)
	if uname != "" {
		uu.Username = uname
	} else {
		uu.Username = origUsr.Username
	}

	email := core.CleanString(uu.Email, true /* lower */)
	if email != "" {
		uu.Email = email
	} else {
		uu.Email = origUsr.Email
	}

	if err := core.Validate.Struct(uu); err != nil {
		return err
	}
	return svc.CheckUniqueness(uu.Username, uu.Email, origUsr)
}

type ResetUserPassword struct {
	Token           string `json:"token,omitempty" validate:"required"`
	UID             string `json:"uid,omitempty" validate:"required"`
	Password        string `json:"password,omitempty" validate:"required"`
	PasswordConfirm string `json:"password_confirm,omitempty" validate:"required,eqfield=Password"`
}

func (rp ResetUserPassword) Validate() error { return core.Validate.Struct(rp) }

type QueryFilter struct {
	Search      string    `query:"search"`
	Roles       []string  `query:"role"`
	IsActive    *bool     `query:"is_active"`
	CreatedFrom time.Time `query:"created_from"`
	CreatedTo   time.Time `query:"created_to"`
}

func (qf *QueryFilter) IsEmpty() bool {
	return qf.Search == "" && qf.Roles == nil && qf.IsActive == nil && qf.CreatedFrom.IsZero() && qf.CreatedTo.IsZero()
}

func (qf *QueryFilter) Clean() {
	qf.Search = core.CleanString(qf.Search)
}
