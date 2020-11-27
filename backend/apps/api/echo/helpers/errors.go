package helpers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/trezcool/masomo/backend/core"
)

var (
	errUnauthorized         = echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	errAuthenticationFailed = echo.NewHTTPError(http.StatusBadRequest, "authentication failed")
	errAccountDeactivated   = echo.NewHTTPError(http.StatusForbidden, "account deactivated")
	errRefreshExpired       = echo.NewHTTPError(http.StatusForbidden, "refresh has expired")
	ErrHttpForbidden        = echo.NewHTTPError(http.StatusForbidden, "permission denied")
	ErrHttpNotFound         = echo.NewHTTPError(http.StatusNotFound, "not found")
	errTokenSigningFailed   = errors.New("failed to sign token")
)

func AppHTTPErrorHandler(err error, c echo.Context) {
	var code int
	var message interface{}

	switch err := err.(type) {
	case *echo.HTTPError:
		if err == middleware.ErrJWTMissing {
			code = http.StatusUnauthorized
			message = err.Message
			break
		}
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
			fldErrs[vErr.Field()] = vErr.Translate(core.Translator)
		}
		code = http.StatusBadRequest
		message = fldErrs
	case *core.ValidationError:
		if err.Fields != nil {
			fldErrs := make(map[string]string)
			for _, fErr := range err.Fields {
				fldErrs[fErr.Field] = fErr.Error
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
		if err != nil {
			c.Echo().Logger.Error(err)
		}
	}
}
