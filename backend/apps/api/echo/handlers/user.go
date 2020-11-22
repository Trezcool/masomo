package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/backend/apps/api/echo/helpers"
	"github.com/trezcool/masomo/backend/business/user"
	"github.com/trezcool/masomo/backend/business/utils"
)

var (
	usrNotFoundInCtxErr  = errors.New("user object not found in echo.Context")
	noPermsToSetRolesErr = "not enough rights to set these roles"
)

type userApi struct {
	service *user.Service
}

func RegisterUserAPI(g *echo.Group, jwt echo.MiddlewareFunc, svc *user.Service) {
	api := userApi{service: svc}

	ug := g.Group("/users")

	// un-authed endpoints
	ug.POST("/login", api.userLogin) // TODO: access attempt
	ug.POST("/password-reset", api.userResetPassword)
	ug.POST("/password-reset-confirm", api.userConfirmPasswordReset)

	// authed endpoints
	ag := ug.Group("", jwt)
	ag.POST("/register", api.userCreate, helpers.AdminMiddleware())
	ag.GET("", api.userQuery, helpers.AdminMiddleware())
	ag.DELETE("", api.userDestroyMultiple, helpers.AdminMiddleware())
	ag.GET("/roles", api.userQueryRoles, helpers.AdminMiddleware())

	// detail endpoints
	dg := ag.Group("/:id", ctxUserOrAdminMiddleware(api.service))
	dg.GET("", api.userRetrieve)
	dg.PUT("", api.userUpdate)
	dg.DELETE("", api.userDestroy, helpers.AdminMiddleware())
}

// Handlers

func (api *userApi) userCreate(ctx echo.Context) error {
	data := new(user.NewUser)
	if err := ctx.Bind(data); err != nil {
		return err
	}
	if err := data.Validate(api.service); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	ctxUsr, err := helpers.GetContextUser(ctx, api.service)
	if err != nil {
		return err
	}
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return utils.NewValidationError(nil, utils.FieldError{Field: "roles", Error: noPermsToSetRolesErr})
	}

	usr, err := api.service.Create(*data)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, usr)
}

func (api *userApi) userLogin(ctx echo.Context) error {
	data := new(LoginRequest)
	if err := ctx.Bind(data); err != nil {
		return err
	}
	if err := data.Validate(); err != nil {
		return err
	}

	claims, err := helpers.Authenticate(data.Username, data.Password, api.service)
	if err != nil {
		return err
	}
	token, err := helpers.GenerateToken(claims)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, LoginResponse{Token: token})
}

func (api *userApi) userResetPassword(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "user.userResetPassword")
} // TODO

func (api *userApi) userConfirmPasswordReset(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "user.userConfirmPasswordReset")
} // TODO

func (api *userApi) userQuery(ctx echo.Context) error {
	users, err := api.service.QueryAll()
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, users)
}

func (api *userApi) userRetrieve(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return usrNotFoundInCtxErr
	}
	return ctx.JSON(http.StatusOK, usr)
}

func (api *userApi) userUpdate(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return usrNotFoundInCtxErr
	}

	data := new(user.UpdateUser)
	if err := ctx.Bind(data); err != nil {
		return err
	}

	ctxUsr, err := helpers.GetContextUser(ctx, api.service)
	if err != nil {
		return err
	}
	if !ctxUsr.IsAdmin() {
		// user cannot edit other users
		if usr.ID != ctxUsr.ID {
			return helpers.ForbiddenHttpErr
		}

		// `IsActive` and `Roles` can only be changed by admin
		// `Username` and `Email` can only be changed by admin for now
		if data.IsActive != nil || data.Roles != nil || data.Username != "" || data.Email != "" {
			return helpers.ForbiddenHttpErr
		}
	}

	if err := data.Validate(usr, api.service); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return utils.NewValidationError(nil, utils.FieldError{Field: "roles", Error: noPermsToSetRolesErr})
	}

	usr, err = api.service.Update(usr.ID, *data)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, usr)
}

func (api *userApi) userDestroy(ctx echo.Context) error {
	usr, ok := ctx.Get("object").(user.User)
	if !ok {
		return usrNotFoundInCtxErr
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := helpers.GetContextUser(ctx, api.service)
	if err != nil {
		return err
	}
	if usr.ID == ctxUsr.ID {
		return helpers.ForbiddenHttpErr
	}

	if err := api.service.Delete(usr.ID); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (api *userApi) userDestroyMultiple(ctx echo.Context) error {
	data := new(DestroyMultipleRequest)
	if err := ctx.Bind(data); err != nil {
		return err
	}
	if data.IDs == nil {
		return ctx.NoContent(http.StatusNoContent)
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := helpers.GetContextUser(ctx, api.service)
	if err != nil {
		return err
	}
	sort.Ints(data.IDs)
	if i := sort.SearchInts(data.IDs, ctxUsr.ID); i < len(data.IDs) {
		if match := data.IDs[i]; ctxUsr.ID == match {
			return helpers.ForbiddenHttpErr
		}
	}

	if err := api.service.Delete(data.IDs...); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (api *userApi) userQueryRoles(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, user.Roles)
}

func ctxUserOrAdminMiddleware(svc *user.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if id, err := strconv.Atoi(ctx.Param("id")); err == nil {
				ctxUsr, err := helpers.GetContextUser(ctx, svc)
				if err != nil {
					return err
				}

				if id == ctxUsr.ID || ctxUsr.IsAdmin() {
					usr, err := svc.GetByID(id)
					if err == nil {
						ctx.Set("object", usr)
						return next(ctx)
					} else if err != user.NotFoundErr {
						return err
					}
				}
			}
			return helpers.NotFoundHttpErr
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

	DestroyMultipleRequest struct {
		IDs []int `query:"id"`
	}
)

func (lr *LoginRequest) Validate() error {
	lr.Username = utils.CleanString(lr.Username, true)
	return utils.Validate.Struct(lr)
}
