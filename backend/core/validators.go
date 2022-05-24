package core

import (
	"reflect"
	"regexp"
	"strings"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	// custom validation tags & texts
	alphaNumUnderTag   = "alphanum_"
	alphaNumUnderText  = "only alphanumeric characters and underscores are allowed"
	alphaNumUnderRegex = regexp.MustCompile(`^[\w\s]+$`)

	requiredTag     = "required"
	requiredWithTag = "required_with"
	requiredText    = "this field is required"
)

// InitValidators instantiates the validator for use.
func InitValidators(validate *validator.Validate, translator ut.Translator) {
	_ = en_translations.RegisterDefaultTranslations(validate, translator)

	// Use JSON tag names for errors instead of Go struct names.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// register custom validators
	_ = validate.RegisterValidation(alphaNumUnderTag, alphaNumUnderValidation)
	RegisterCustomTranslation(validate, translator, alphaNumUnderTag, alphaNumUnderText)

	RegisterCustomTranslation(validate, translator, requiredTag, requiredText, true)
	RegisterCustomTranslation(validate, translator, requiredWithTag, requiredText, true)
}

// RegisterCustomTranslation registers a custom translation for the specified validation tag.
func RegisterCustomTranslation(validate *validator.Validate, translator ut.Translator, tag, text string, override ...bool) {
	var ovrd bool
	if len(override) > 0 {
		ovrd = override[0]
	}
	_ = validate.RegisterTranslation(
		tag, translator,
		func(t ut.Translator) error { return t.Add(tag, text, ovrd) },
		func(t ut.Translator, fe validator.FieldError) string {
			s, _ := t.T(tag, fe.Field())
			return s
		},
	)
}

// Custom Global Validators

// alphaNumUnderValidation only allows alphanumeric characters and underscores.
func alphaNumUnderValidation(fl validator.FieldLevel) bool {
	return alphaNumUnderRegex.MatchString(fl.Field().String())
}
