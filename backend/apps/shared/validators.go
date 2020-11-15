package shared

import (
	"reflect"
	"sort"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"

	"github.com/trezcool/masomo/backend/apps/user"
)

var (
	Validate   *validator.Validate
	Translator ut.Translator

	// custom validation tags
	notBlankTag        = "notblank"
	usernameOrEmailTag = "username_or_email"
	allRolesTag        = "all_roles"
)

// Instantiate the validator for use.
func init() {
	Validate = validator.New()

	// Register the english error messages for validation errors.
	_en := en.New()
	uni := ut.New(_en, _en)
	Translator, _ = uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(Validate, Translator)

	// Use JSON tag names for errors instead of Go struct names.
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// register custom validators
	_ = Validate.RegisterValidation(notBlankTag, notBlankValidation)
	_ = Validate.RegisterValidation(allRolesTag, validateAllRoles)
	Validate.RegisterStructValidation(newUserStructValidation, user.NewUser{})

	registerCustomValidationsTranslations(notBlankTag, usernameOrEmailTag, allRolesTag)
}

// registerCustomValidationsTranslations registers error messages for custom struct validations.
// a validator.RegisterTranslationsFunc is required for registering the Translator,
// but it has already been registered as the default translation.
// so a noop func is passed to bypass this requirement.
func registerCustomValidationsTranslations(tags ...string) {
	registerFn := func(ut.Translator) error { return nil }
	for _, tag := range tags {
		_ = Validate.RegisterTranslation(tag, Translator, registerFn, translateCustomValidationErrs)
	}
}

func translateCustomValidationErrs(_ ut.Translator, fe validator.FieldError) string {
	switch fe.Tag() {
	case notBlankTag:
		return "this field cannot be blank"
	case usernameOrEmailTag:
		return "one of username or email is required"
	case allRolesTag:
		return "invalid roles"
	default:
		return ""
	}
}

// Custom Validators

func notBlankValidation(fl validator.FieldLevel) bool {
	if str, ok := fl.Field().Interface().(string); ok {
		return strings.TrimSpace(str) != ""
	}
	return false
}

// newUserStructValidation does user.NewUser's struct level validation
func newUserStructValidation(sl validator.StructLevel) {
	if nu, ok := sl.Current().Interface().(user.NewUser); ok {
		// one of Username or Email is required
		uname := strings.TrimSpace(nu.Username)
		email := strings.TrimSpace(nu.Email)
		if len(uname) == 0 && len(email) == 0 {
			sl.ReportError(nu.Username, "username", "Username", usernameOrEmailTag, "")
			sl.ReportError(nu.Email, "email", "Email", usernameOrEmailTag, "")
		}
	}
}

// validateAllRoles checks that provided user roles are all in user.AllRoles
func validateAllRoles(fl validator.FieldLevel) bool {
	if roles, ok := fl.Field().Interface().([]string); ok {
		sort.Strings(user.AllRoles)
		for _, role := range roles {
			if idx := sort.SearchStrings(user.AllRoles, role); idx < len(user.AllRoles) {
				if match := user.AllRoles[idx]; role != match {
					return false
				}
			}
		}
		return true
	}
	return false
}
