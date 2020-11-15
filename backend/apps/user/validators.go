package user

import (
	"sort"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"github.com/trezcool/masomo/backend/apps/utils"
)

var (
	allRolesTag        = "all_roles"
	usernameOrEmailTag = "username_or_email"
)

// register custom validators
func init() {
	_ = utils.Validate.RegisterValidation(allRolesTag, allRolesValidation)
	utils.Validate.RegisterStructValidation(newUserStructValidation, NewUser{})

	utils.RegisterCustomValidationsTranslations(
		translateCustomValidationErrs,
		allRolesTag,
		usernameOrEmailTag,
	)
}

func translateCustomValidationErrs(_ ut.Translator, fe validator.FieldError) string {
	switch fe.Tag() {
	case allRolesTag:
		return "invalid roles"
	case usernameOrEmailTag:
		return "one of username or email is required"
	default:
		return ""
	}
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
