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

	// custom validation tags & texts
	alphaNumUnderTag   = "alphanum_"
	alphaNumUnderText  = "only alphanumeric characters and underscores are allowed"
	alphaNumUnderRegex = regexp.MustCompile("^[\\w\\s]+$")
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
	RegisterCustomTranslation(alphaNumUnderTag, alphaNumUnderText)
}

// RegisterCustomTranslation registers a custom translation for the specified validation tag.
func RegisterCustomTranslation(tag, text string, override ...bool) {
	var ovrd bool
	if len(override) > 0 {
		ovrd = override[0]
	}
	_ = Validate.RegisterTranslation(
		tag, Translator,
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
