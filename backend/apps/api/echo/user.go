package echoapi

import (
	"net/http"
	"sort"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

var (
	errUsrNotFoundInCtx  = errors.New("user object not found in echo.Context")
	errNoPermsToSetRoles = "not enough rights to set these roles"
)

type userApi struct {
	svc        user.ServiceInterface
	validate   *validator.Validate
	translator ut.Translator
}

func registerUserAPI(
	g *echo.Group,
	jwt echo.MiddlewareFunc,
	svc user.ServiceInterface,
	validate *validator.Validate,
	translator ut.Translator,
) {
	api := userApi{
		svc:        svc,
		validate:   validate,
		translator: translator,
	}

	ug := g.Group("/users")

	// un-authed endpoints
	// TODO: access attempt
	// TODO: no concurrent sessions
	// TODO: rate limit `/password-reset` & `/password-reset-confirm`
	ug.POST("/login", api.login)
	ug.POST("/password-reset", api.resetPassword)
	ug.POST("/password-reset-confirm", api.confirmPasswordReset)

	// authed endpoints
	ag := ug.Group("", jwt)
	ag.POST("/token-refresh", api.refreshToken)
	ag.POST("/register", api.create, adminMiddleware())
	ag.GET("", api.query, adminMiddleware())
	ag.DELETE("", api.destroyMultiple, adminMiddleware())
	ag.GET("/roles", api.queryRoles, adminMiddleware())

	// detail endpoints
	dg := ag.Group("/:id", ctxUserOrAdminMiddleware(api.svc))
	dg.GET("", api.retrieve)
	dg.PUT("", api.update)
	dg.DELETE("", api.destroy, adminMiddleware())
}

// Handlers

func (api *userApi) create(ctx echo.Context) error {
	var data user.NewUser
	if err := ctx.Bind(&data); err != nil {
		return errors.Wrap(err, "binding to NewUser")
	}
	if err := data.Validate(api.validate, api.svc); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return errors.Wrap(err, "getting context user")
	}
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return core.NewValidationError(nil, core.FieldError{Field: "roles", Error: errNoPermsToSetRoles})
	}

	usr, err := api.svc.Create(data)
	if err != nil {
		return errors.Wrap(err, "creating user")
	}

	return ctx.JSON(http.StatusCreated, usr)
}

func (api *userApi) login(ctx echo.Context) error {
	var data LoginRequest
	if err := ctx.Bind(&data); err != nil {
		return errors.Wrap(err, "binding to LoginRequest")
	}
	if err := data.Validate(api.validate); err != nil {
		return err
	}

	claims, err := authenticate(data.Username, data.Password, api.svc)
	if err != nil {
		if errors.Cause(err) == user.ErrNotFound {
			return core.NewValidationError(errors.New("invalid credentials"))
		}
		return errors.Wrap(err, "authenticating")
	}
	token, err := GenerateToken(claims)
	if err != nil {
		return errors.Wrap(err, "generating token")
	}

	return ctx.JSON(http.StatusOK, LoginResponse{Token: token})
}

func (api *userApi) resetPassword(ctx echo.Context) error {
	var data PasswordResetRequest
	if err := ctx.Bind(&data); err != nil {
		return errors.Wrap(err, "binding to PasswordResetRequest")
	}
	if err := data.Validate(api.validate); err != nil {
		return err
	}

	if err := api.svc.RequestPasswordReset(data.Email); !(err == nil || errors.Cause(err) == user.ErrNotFound) {
		// do not return errors to attackers
		ctx.Logger().Errorf("%+v", errors.Wrap(err, "requesting password reset"))
	}
	return ctx.JSON(http.StatusOK, SuccessResponse{
		Success: "If the email address supplied is associated with an active account on this system, " +
			"an email will arrive in your inbox shortly with instructions to reset your password.",
	})
}

func (api *userApi) confirmPasswordReset(ctx echo.Context) error {
	var data user.ResetUserPassword
	if err := ctx.Bind(&data); err != nil {
		return errors.Wrap(err, "binding to ResetUserPassword")
	}
	if err := data.Validate(api.validate); err != nil {
		return err
	}

	if err := api.svc.ResetPassword(data); err != nil {
		return errors.Wrap(err, "resetting password")
	}
	return ctx.JSON(http.StatusOK, SuccessResponse{Success: "Password has been reset with the new password."})
}

func (api *userApi) query(ctx echo.Context) error {
	filter := new(user.QueryFilter)
	if err := ctx.Bind(filter); err != nil {
		return ctx.JSON(http.StatusOK, []user.User{})
	}
	filter.Clean()
	ordering := new(Ordering)
	ordering.Bind(ctx)

	users, err := api.svc.Query(filter, ordering.Orderings)
	if err != nil {
		return errors.Wrap(err, "querying users")
	}
	if users == nil {
		users = []user.User{}
	}
	return ctx.JSON(http.StatusOK, users)
}

func (api *userApi) retrieve(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return errors.Wrap(errUsrNotFoundInCtx, "retrieving object from context")
	}
	return ctx.JSON(http.StatusOK, usr)
}

func (api *userApi) update(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return errors.Wrap(errUsrNotFoundInCtx, "retrieving object from context")
	}

	var data user.UpdateUser
	if err := ctx.Bind(data); err != nil {
		return errors.Wrap(err, "binding to UpdateUser")
	}

	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return errors.Wrap(err, "getting context user")
	}
	if !ctxUsr.IsAdmin() {
		// `IsActive` and `Roles` can only be changed by admin
		// `Username` and `Email` can only be changed by admin for now
		if data.IsActive != nil || data.Roles != nil || data.Username != "" || data.Email != "" {
			return errHttpForbidden
		}
	}

	if err := data.Validate(usr, api.validate, api.svc); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return core.NewValidationError(nil, core.FieldError{Field: "roles", Error: errNoPermsToSetRoles})
	}

	usr, err = api.svc.Update(usr.ID, data)
	if err != nil {
		return errors.Wrap(errUsrNotFoundInCtx, "updating user")
	}

	return ctx.JSON(http.StatusOK, usr)
}

func (api *userApi) destroy(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return errors.Wrap(errUsrNotFoundInCtx, "retrieving object from context")
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return errors.Wrap(err, "getting context user")
	}
	if usr.ID == ctxUsr.ID {
		return errHttpForbidden
	}

	// TODO: ctxUser cannot delete a User with a max role > theirs

	if err := api.svc.Delete(usr.ID); err != nil {
		return errors.Wrap(errUsrNotFoundInCtx, "deleting user")
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (api *userApi) destroyMultiple(ctx echo.Context) error {
	var query DestroyMultipleRequest
	if err := ctx.Bind(&query); err != nil {
		return errors.Wrap(err, "binding to DestroyMultipleRequest")
	}
	if query.IDs == nil {
		return ctx.NoContent(http.StatusNoContent)
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := getContextUser(ctx, api.svc)
	if err != nil {
		return errors.Wrap(err, "getting context user")
	}
	sort.Strings(query.IDs)
	if i := sort.SearchStrings(query.IDs, ctxUsr.ID); i < len(query.IDs) {
		if match := query.IDs[i]; ctxUsr.ID == match {
			return errHttpForbidden
		}
	}

	// TODO: ctxUser cannot delete a User with a max role > theirs

	if err := api.svc.Delete(query.IDs...); err != nil {
		return errors.Wrap(errUsrNotFoundInCtx, "deleting users")
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (api *userApi) queryRoles(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, user.Roles)
}

func (api *userApi) refreshToken(ctx echo.Context) error {
	token, err := refreshToken(ctx, api.svc)
	if err != nil {
		return errors.Wrap(err, "refreshing token")
	}
	return ctx.JSON(http.StatusOK, LoginResponse{Token: token})
}

func ctxUserOrAdminMiddleware(svc user.ServiceInterface) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			ctxUsr, err := getContextUser(ctx, svc)
			if err != nil {
				return errors.Wrap(err, "getting context user")
			}

			if ctx.Param("id") == ctxUsr.ID || ctxUsr.IsAdmin() {
				if usr, err := svc.GetByID(ctx.Param("id")); err == nil {
					ctx.Set("object", usr)
					return next(ctx)
				} else if errors.Cause(err) != user.ErrNotFound {
					return errors.Wrap(err, "finding user by ID")
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
		IDs []string `query:"id"`
	}
)

func (lr *LoginRequest) Validate(validate *validator.Validate) error {
	lr.Username = core.CleanString(lr.Username, true /* lower */)
	return validate.Struct(lr)
}

func (pr *PasswordResetRequest) Validate(validate *validator.Validate) error {
	pr.Email = core.CleanString(pr.Email, true /* lower */)
	return validate.Struct(pr)
}
