package helpers

import "github.com/labstack/echo/v4"

func AdminMiddleware(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			claims, err := getContextClaims(ctx)
			if err != nil {
				return err
			}
			if claims.IsAdmin && contextHasAnyRole(ctx, roles) {
				return next(ctx)
			}
			return ForbiddenHttpErr
		}
	}
}
