package user

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/pmezard/go-difflib/difflib"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/fs"
)

var (
	allRolesTag  = "allroles"
	allRolesText = "invalid roles"

	usernameOrEmailTag  = "username_or_email"
	usernameOrEmailText = "one of username or email is required"

	// password policy
	pwdMinLen     = 8
	pwdMinLenTag  = "pwdminlen"
	pwdMinLenText = fmt.Sprintf("password must contain at least %d characters", pwdMinLen)

	pwdNoSpaceTag  = "pwdnospace"
	pwdNoSpaceText = "password must not contain whitespace"

	pwdNotAllNumTag  = "pwdnotallnum"
	pwdNotAllNumText = "password cannot be entirely numeric"

	pwdComplexityTag  = "pwdcplx"
	pwdComplexityText = "password must contain at least 1 uppercase character, 1 lowercase character, 1 digit and 1 special character"
	specialRegex      = regexp.MustCompile(`[^A-Za-z0-9]`)

	pwdMaxSim      = .7
	pwdAttrSimTag  = "pwdtoosim"
	pwdAttrSimText = "password cannot be similar to user attributes"

	pwdNoCommonTag  = "pwdnocommon"
	pwdNoCommonText = "password is too common"
	commonPasswords []string
)

// InitValidators registers validators
func InitValidators(validate *validator.Validate, translator ut.Translator) {
	_ = validate.RegisterValidation(allRolesTag, allRolesValidation)
	core.RegisterCustomTranslation(validate, translator, allRolesTag, allRolesText)

	validate.RegisterStructValidation(userStructValidation, NewUser{})
	validate.RegisterStructValidation(userStructValidation, UpdateUser{})
	validate.RegisterStructValidation(userStructValidation, ResetUserPassword{})
	core.RegisterCustomTranslation(validate, translator, usernameOrEmailTag, usernameOrEmailText)
	core.RegisterCustomTranslation(validate, translator, pwdMinLenTag, pwdMinLenText)
	core.RegisterCustomTranslation(validate, translator, pwdNoSpaceTag, pwdNoSpaceText)
	core.RegisterCustomTranslation(validate, translator, pwdNotAllNumTag, pwdNotAllNumText)
	core.RegisterCustomTranslation(validate, translator, pwdComplexityTag, pwdComplexityText)
	core.RegisterCustomTranslation(validate, translator, pwdAttrSimTag, pwdAttrSimText)
	core.RegisterCustomTranslation(validate, translator, pwdNoCommonTag, pwdNoCommonText)
}

func LoadCommonPasswords(logger core.Logger) {
	commonPasswords = make([]string, 0, 19727) // 19727: number of total pwds in /assets/common-passwords.txt.gz
	pwdAssetPath := "assets/common-passwords.txt.gz"

	file, err := appfs.FS.Open(pwdAssetPath)
	if err != nil {
		logger.Error(errors.Wrap(err, "opening "+pwdAssetPath).Error(), err)
		return
	}
	defer func() { _ = file.Close() }()

	gzRdr, err := gzip.NewReader(file)
	if err != nil {
		logger.Error(errors.Wrap(err, "creating gzip reader").Error(), err)
		return
	}

	scanner := bufio.NewScanner(gzRdr)
	for scanner.Scan() {
		commonPasswords = append(commonPasswords, strings.TrimSpace(scanner.Text()))
	}
	if scanner.Err() != nil {
		logger.Error(errors.Wrap(err, "scanning "+pwdAssetPath).Error(), err)
	}

	sort.Strings(commonPasswords)
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

// userStructValidation does struct level validation on NewUser, UpdateUser and ResetUserPassword structs.
func userStructValidation(sl validator.StructLevel) {
	switch usr := sl.Current().Interface().(type) {
	case NewUser:
		validateUsernameAndEmail(usr, sl)
		validatePassword(usr.Password, usr.Name, usr.Username, usr.Email, sl)
	case UpdateUser:
		if usr.Password != "" {
			validatePassword(usr.Password, usr.Name, usr.Username, usr.Email, sl)
		}
	case ResetUserPassword:
		validatePassword(usr.Password, "", "", "", sl)
	}
}

// validateUsernameAndEmail checks that one of Username or Email is provided
func validateUsernameAndEmail(nu NewUser, sl validator.StructLevel) {
	if len(nu.Username) == 0 && len(nu.Email) == 0 {
		sl.ReportError(nu.Username, "username", "Username", usernameOrEmailTag, "")
		sl.ReportError(nu.Email, "email", "Email", usernameOrEmailTag, "")
	}
}

// validatePassword applies the password policy to provided password:
// - minLen: 8
// - no whitespace
// - no all numeric
// - complexity: 1 upper, 1 lower, 1 digit, 1 special
// - no user attrs similarity
// - no common password
func validatePassword(pwd, name, uname, email string, sl validator.StructLevel) {
	reportErr := func(tag string) {
		sl.ReportError(pwd, "password", "Password", tag, "")
	}

	var (
		digitCount                             int
		hasUpper, hasLower, hasDig, hasSpecial bool
	)

	// - minLen: 8
	pwdLen := len(pwd)
	if pwdLen < 8 {
		reportErr(pwdMinLenTag)
		return
	}
	for _, char := range pwd {
		// - no whitespace
		if unicode.IsSpace(char) {
			reportErr(pwdNoSpaceTag)
			return
		}
		if unicode.IsDigit(char) {
			digitCount++
		}
		if !hasUpper && unicode.IsUpper(char) {
			hasUpper = true
		}
		if !hasLower && unicode.IsLower(char) {
			hasLower = true
		}
	}

	// - not all numeric
	if digitCount == pwdLen {
		reportErr(pwdNotAllNumTag)
		return
	}

	// - complexity: 1 upper, 1 lower, 1 digit & 1 special
	hasDig = digitCount > 0
	hasSpecial = specialRegex.MatchString(pwd)
	if !(hasUpper && hasLower && hasDig && hasSpecial) {
		reportErr(pwdComplexityTag)
		return
	}

	// - no user attrs similarity
	getRatio := func(pass, usrAttr string) float64 {
		if usrAttr == "" {
			return 0
		}
		return difflib.NewMatcher(strings.Split(pass, ""), strings.Split(usrAttr, "")).QuickRatio()
	}
	if getRatio(pwd, name) >= pwdMaxSim ||
		getRatio(pwd, uname) >= pwdMaxSim ||
		getRatio(pwd, email) >= pwdMaxSim {
		reportErr(pwdAttrSimTag)
		return
	}

	// - no common passwords
	lpwd := strings.ToLower(pwd)
	if i := sort.SearchStrings(commonPasswords, lpwd); i < len(commonPasswords) {
		if match := commonPasswords[i]; lpwd == match {
			reportErr(pwdNoCommonTag)
			return
		}
	}
}
