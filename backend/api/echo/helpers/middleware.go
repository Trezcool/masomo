package helpers

import "github.com/labstack/echo/v4"

func AdminMiddleware(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, err := getContextClaims(c)
			if err != nil {
				return err
			}
			if claims.IsAdmin && contextHasAnyRole(c, roles) {
				return next(c)
			}
			return ForbiddenHttpErr
		}
	}
}
