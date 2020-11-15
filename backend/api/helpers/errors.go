package helpers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/backend/apps/shared"
)

var (
	unauthorizedErr         = echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	authenticationFailedErr = echo.NewHTTPError(http.StatusBadRequest, "authentication failed")
	accountDeactivatedErr   = echo.NewHTTPError(http.StatusForbidden, "account deactivated")
	ForbiddenHttpErr        = echo.NewHTTPError(http.StatusForbidden, "permission denied")
	NotFoundHttpErr         = echo.NewHTTPError(http.StatusNotFound, "not found")
	tokenSigningError       = errors.New("failed to sign token")
)

// FieldError is used to indicate an error with a specific request field.
type FieldError struct {
	field string
	error string
}

func NewFieldError(field, err string) FieldError {
	return FieldError{field, err}
}

type badRequestError struct {
	err    error
	code   int
	fields []FieldError
}

// NewBadRequestError wraps a provided error with an http.StatusBadRequest code.
func NewBadRequestError(err error, flds ...FieldError) error {
	return &badRequestError{err, http.StatusBadRequest, flds}
}

func (err *badRequestError) Error() string {
	if err.err == nil {
		return ""
	}
	return err.err.Error()
}

func AppHTTPErrorHandler(err error, c echo.Context) {
	var code int
	var message interface{}

	switch err := err.(type) {
	case *echo.HTTPError:
		if err.Internal != nil {
			if herr, ok := err.Internal.(*echo.HTTPError); ok {
				err = herr
			}
		}
		code = err.Code
		message = err.Message
	case validator.ValidationErrors:
		fldErrs := make(map[string]string)
		for _, vErr := range err {
			fldErrs[vErr.Field()] = vErr.Translate(shared.Translator)
		}
		code = http.StatusBadRequest
		message = fldErrs
	case *badRequestError:
		if err.fields != nil {
			fldErrs := make(map[string]string)
			for _, fErr := range err.fields {
				fldErrs[fErr.field] = fErr.error
			}
			message = fldErrs
		} else {
			message = err.Error()
		}
		code = http.StatusBadRequest
	default: // any other error is a server error
		code = http.StatusInternalServerError
		message = http.StatusText(http.StatusInternalServerError)
	}

	if c.Echo().Debug {
		message = err.Error()
	} else if m, ok := message.(string); ok {
		message = echo.Map{"error": m}
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead { // Issue #608
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, message)
		}
		if code >= http.StatusInternalServerError {
			c.Echo().Logger.Error(err)
		}
	}
}
