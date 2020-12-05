package echoapi

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
)

var (
	errUsrNotFoundInCtx  = errors.New("user object not found in echo.Context")
	errNoPermsToSetRoles = "not enough rights to set these roles"
)

type userApi struct {
	svc user.Service
}

func registerUserAPI(g *echo.Group, jwt echo.MiddlewareFunc, svc user.Service) {
	api := userApi{svc: svc}

	ug := g.Group("/users")

	// un-authed endpoints
	// TODO: access attempt
	// TODO: no concurrent sessions
	// TODO: rate limit `/password-reset` & `/password-reset-confirm`
	ug.POST("/login", api.userLogin)
	ug.POST("/password-reset", api.userResetPassword)
	ug.POST("/password-reset-confirm", api.userConfirmPasswordReset)

	// authed endpoints
	ag := ug.Group("", jwt)
	ag.POST("/token-refresh", api.userRefreshToken)
	ag.POST("/register", api.userCreate, adminMiddleware())
	ag.GET("", api.userQuery, adminMiddleware())
	ag.DELETE("", api.userDestroyMultiple, adminMiddleware())
	ag.GET("/roles", api.userQueryRoles, adminMiddleware())

	// detail endpoints
	dg := ag.Group("/:id", ctxUserOrAdminMiddleware(api.svc))
	dg.GET("", api.userRetrieve)
	dg.PUT("", api.userUpdate)
	dg.DELETE("", api.userDestroy, adminMiddleware())
}

// Handlers

func (api *userApi) userCreate(ctx echo.Context) error {
	var data user.NewUser
	if err := ctx.Bind(&data); err != nil {
		return err
	}
	if err := data.Validate(api.svc); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return err
	}
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return core.NewValidationError(nil, core.FieldError{Field: "roles", Error: errNoPermsToSetRoles})
	}

	usr, err := api.svc.Create(data)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, usr)
}

func (api *userApi) userLogin(ctx echo.Context) error {
	var data LoginRequest
	if err := ctx.Bind(&data); err != nil {
		return err
	}
	if err := data.Validate(); err != nil {
		return err
	}

	claims, err := authenticate(data.Username, data.Password, api.svc)
	if err != nil {
		return err
	}
	token, err := GenerateToken(claims)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, LoginResponse{Token: token})
}

func (api *userApi) userResetPassword(ctx echo.Context) error {
	var data PasswordResetRequest
	if err := ctx.Bind(&data); err != nil {
		return err
	}
	if err := data.Validate(); err != nil {
		return err
	}

	if err := api.svc.RequestPasswordReset(data.Email); !(err == nil || err == user.ErrNotFound) {
		// do not return errors to attackers
		ctx.Logger().Error(err)
	}
	return ctx.JSON(http.StatusOK, SuccessResponse{
		Success: "If the email address supplied is associated with an active account on this system, " +
			"an email will arrive in your inbox shortly with instructions to reset your password.",
	})
}

func (api *userApi) userConfirmPasswordReset(ctx echo.Context) error {
	var data user.ResetUserPassword
	if err := ctx.Bind(&data); err != nil {
		return err
	}
	if err := data.Validate(); err != nil {
		return err
	}

	if err := api.svc.ResetPassword(data); err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, SuccessResponse{Success: "Password has been reset with the new password."})
}

func (api *userApi) userQuery(ctx echo.Context) error {
	var query user.QueryFilter
	if err := ctx.Bind(&query); err != nil {
		return ctx.JSON(http.StatusOK, []user.User{})
	}
	query.Clean()

	if query.IsEmpty() {
		users, err := api.svc.QueryAll()
		if err != nil {
			return err
		}
		if users == nil {
			users = []user.User{}
		}
		return ctx.JSON(http.StatusOK, users)
	}

	users, err := api.svc.Filter(query)
	if err != nil {
		return err
	}
	if users == nil {
		users = []user.User{}
	}
	return ctx.JSON(http.StatusOK, users)
}

func (api *userApi) userRetrieve(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return errUsrNotFoundInCtx
	}
	return ctx.JSON(http.StatusOK, usr)
}

func (api *userApi) userUpdate(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return errUsrNotFoundInCtx
	}

	var data user.UpdateUser
	if err := ctx.Bind(data); err != nil {
		return err
	}

	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return err
	}
	if !ctxUsr.IsAdmin() {
		// user cannot edit other users
		if usr.ID != ctxUsr.ID {
			return errHttpForbidden
		}

		// `IsActive` and `Roles` can only be changed by admin
		// `Username` and `Email` can only be changed by admin for now
		if data.IsActive != nil || data.Roles != nil || data.Username != "" || data.Email != "" {
			return errHttpForbidden
		}
	}

	if err := data.Validate(usr, api.svc); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return core.NewValidationError(nil, core.FieldError{Field: "roles", Error: errNoPermsToSetRoles})
	}

	usr, err = api.svc.Update(usr.ID, data)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, usr)
}

func (api *userApi) userDestroy(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return errUsrNotFoundInCtx
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return err
	}
	if usr.ID == ctxUsr.ID {
		return errHttpForbidden
	}

	if err := api.svc.Delete(usr.ID); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (api *userApi) userDestroyMultiple(ctx echo.Context) error {
	var query DestroyMultipleRequest
	if err := ctx.Bind(&query); err != nil {
		return err
	}
	if query.IDs == nil {
		return ctx.NoContent(http.StatusNoContent)
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return err
	}
	sort.Ints(query.IDs)
	if i := sort.SearchInts(query.IDs, ctxUsr.ID); i < len(query.IDs) {
		if match := query.IDs[i]; ctxUsr.ID == match {
			return errHttpForbidden
		}
	}

	if err := api.svc.Delete(query.IDs...); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (api *userApi) userQueryRoles(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, user.Roles)
}

func (api *userApi) userRefreshToken(ctx echo.Context) error {
	token, err := refreshToken(ctx, api.svc)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, LoginResponse{Token: token})
}

func ctxUserOrAdminMiddleware(svc user.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if id, err := strconv.Atoi(ctx.Param("id")); err == nil {
				ctxUsr, err := getContextUser(ctx, svc)
				if err != nil {
					return err
				}

				if id == ctxUsr.ID || ctxUsr.IsAdmin() {
					usr, err := svc.GetByID(id)
					if err == nil {
						ctx.Set("object", usr)
						return next(ctx)
					} else if err != user.ErrNotFound {
						return err
					}
				}
			}
			return errHttpNotFound
		}
	}
}

type (
	LoginRequest struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	LoginResponse struct {
		Token string `json:"token"`
	}

	PasswordResetRequest struct {
		Email string `json:"email" validate:"required,email"`
	}

	SuccessResponse struct {
		Success string `json:"success"`
	}

	DestroyMultipleRequest struct {
		IDs []int `query:"id"`
	}
)

func (lr *LoginRequest) Validate() error {
	lr.Username = core.CleanString(lr.Username, true /* lower */)
	return core.Validate.Struct(lr)
}

func (pr *PasswordResetRequest) Validate() error {
	pr.Email = core.CleanString(pr.Email, true /* lower */)
	return core.Validate.Struct(pr)
}