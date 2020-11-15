package utils

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	Validate   *validator.Validate
	Translator ut.Translator

	// custom validation tags
	alphaNumUnderTag = "alphanum_"

	alphaNumUnderRegex = regexp.MustCompile("^[a-zA-Z0-9_]+$")
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
	_ = Validate.RegisterValidation(alphaNumUnderTag, alphaNumUnderValidation)

	RegisterCustomValidationsTranslations(
		translateCustomValidationErrs,
		alphaNumUnderTag,
	)
}

// registerCustomValidationsTranslations registers error messages for custom struct validations.
// a validator.RegisterTranslationsFunc is required for registering the Translator,
// but it has already been registered as the default translation.
// so a noop func is passed to bypass this requirement.
func RegisterCustomValidationsTranslations(transFn validator.TranslationFunc, tags ...string) {
	registerFn := func(ut.Translator) error { return nil }
	for _, tag := range tags {
		_ = Validate.RegisterTranslation(tag, Translator, registerFn, transFn)
	}
}

func translateCustomValidationErrs(_ ut.Translator, fe validator.FieldError) string {
	switch fe.Tag() {
	case alphaNumUnderTag:
		return "only alphanumeric characters and underscores are allowed"
	default:
		return ""
	}
}

// Custom Global Validators

// alphaNumUnderValidation only allows alphanumeric characters and underscores.
func alphaNumUnderValidation(fl validator.FieldLevel) bool {
	if str, ok := fl.Field().Interface().(string); ok {
		return alphaNumUnderRegex.MatchString(str)
	}
	return false
}
