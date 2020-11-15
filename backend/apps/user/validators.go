package user

import (
	"sort"

	"github.com/go-playground/validator/v10"

	"github.com/trezcool/masomo/backend/apps/utils"
)

var (
	allRolesTag  = "all_roles"
	allRolesText = "invalid roles"

	usernameOrEmailTag  = "username_or_email"
	usernameOrEmailText = "one of username or email is required"
)

// register custom validators
func init() {
	_ = utils.Validate.RegisterValidation(allRolesTag, allRolesValidation)
	utils.RegisterCustomTranslation(allRolesTag, allRolesText)

	utils.Validate.RegisterStructValidation(newUserStructValidation, NewUser{})
	utils.RegisterCustomTranslation(usernameOrEmailTag, usernameOrEmailText)
}

// Custom Validators

// allRolesValidation checks that provided user roles are all in AllRoles
func allRolesValidation(fl validator.FieldLevel) bool {
	if roles, ok := fl.Field().Interface().([]string); ok {
		sort.Strings(AllRoles)
		for _, role := range roles {
			if idx := sort.SearchStrings(AllRoles, role); idx < len(AllRoles) {
				if match := AllRoles[idx]; role != match {
					return false
				}
			}
		}
		return true
	}
	return false
}

// newUserStructValidation does NewUser's struct level validation
func newUserStructValidation(sl validator.StructLevel) {
	if nu, ok := sl.Current().Interface().(NewUser); ok {
		// one of Username or Email is required
		if len(nu.Username) == 0 && len(nu.Email) == 0 {
			sl.ReportError(nu.Username, "username", "Username", usernameOrEmailTag, "")
			sl.ReportError(nu.Email, "email", "Email", usernameOrEmailTag, "")
		}
	}
}
