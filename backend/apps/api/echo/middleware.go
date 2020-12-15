package echoapi

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func adminMiddleware(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			claims, err := getContextClaims(ctx)
			if err != nil {
				return errors.Wrap(err, "getting context claims")
			}
			if claims.IsAdmin && contextHasAnyRole(ctx, roles) {
				return next(ctx)
			}
			return errHttpForbidden
		}
	}
}
