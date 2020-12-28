package echoapi

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

var (
	errUnauthorized         = echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	errAuthenticationFailed = echo.NewHTTPError(http.StatusBadRequest, "authentication failed")
	errAccountDeactivated   = echo.NewHTTPError(http.StatusForbidden, "account deactivated")
	errRefreshExpired       = echo.NewHTTPError(http.StatusForbidden, "refresh has expired")
	errHttpForbidden        = echo.NewHTTPError(http.StatusForbidden, "permission denied")
	errHttpNotFound         = echo.NewHTTPError(http.StatusNotFound, "not found")
)

// newAppHTTPErrorHandler returns a custom echo.HTTPErrorHandler that knows how to handle our errors.
// signalShutdown is called in order to gracefully shutdown the Server whenever a core.shutdown error is caught.
func newAppHTTPErrorHandler(logger core.Logger, signalShutdown func()) echo.HTTPErrorHandler {
	return func(err error, ctx echo.Context) {
		var code int
		var message interface{}

		switch origErr := errors.Cause(err).(type) {
		case *echo.HTTPError:
			if origErr == middleware.ErrJWTMissing {
				code = http.StatusUnauthorized
				message = origErr.Message
				break
			}
			if origErr.Internal != nil {
				if herr, ok := origErr.Internal.(*echo.HTTPError); ok {
					origErr = herr
				}
			}
			code = origErr.Code
			message = origErr.Message
		case validator.ValidationErrors:
			fldErrs := make(map[string]string, len(origErr))
			for _, vErr := range origErr {
				fldErrs[vErr.Field()] = vErr.Translate(core.Translator)
			}
			code = http.StatusBadRequest
			message = fldErrs
		case *core.ValidationError:
			if origErr.Fields != nil {
				fldErrs := make(map[string]string, len(origErr.Fields))
				for _, fErr := range origErr.Fields {
					fldErrs[fErr.Field] = fErr.Error
				}
				message = fldErrs
			} else {
				message = origErr.Error()
			}
			code = http.StatusBadRequest
		default: // any other error is a server error
			code = http.StatusInternalServerError
			msg := http.StatusText(http.StatusInternalServerError)
			message = msg

			var usr user.User
			if claims, cErr := getContextClaims(ctx); cErr == nil {
				usr.ID = claims.Subject
				usr.Username = claims.Username
				usr.Email = claims.Email
			}
			logger.Error(msg, errors.Wrap(err, msg), usr)

			// shutting down...
			if core.IsShutdown(err) {
				signalShutdown()
			}
		}

		if ctx.Echo().Debug {
			message = err.Error()
		} else if m, ok := message.(string); ok {
			message = echo.Map{"error": m}
		}

		// Send response
		if !ctx.Response().Committed {
			if ctx.Request().Method == http.MethodHead { // Issue #608
				err = ctx.NoContent(code)
			} else {
				err = ctx.JSON(code, message)
			}
			if err != nil {
				ctx.Echo().Logger.Error(err)
			}
		}
	}
}
