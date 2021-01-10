package boiledrepos

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/storage/database/sqlboiler/models"
)

type userRepository struct {
	exec core.DBExecutor
}

var _ user.Repository = (*userRepository)(nil) // interface compliance check

func NewUserRepository(exec core.DBExecutor) *userRepository {
	return &userRepository{exec: exec}
}

func (repo userRepository) getExec(svcExec []core.DBExecutor) core.DBExecutor {
	if len(svcExec) > 0 {
		return svcExec[0]
	}
	return repo.exec
}

func (repo userRepository) boil(usr user.User) *models.User {
	u := &models.User{
		Name:         null.NewString(usr.Name, usr.Name != ""),
		Username:     null.NewString(usr.Username, usr.Username != ""),
		Email:        null.NewString(usr.Email, usr.Email != ""),
		IsActive:     null.BoolFromPtr(usr.IsActive),
		Roles:        usr.Roles,
		PasswordHash: null.BytesFrom(usr.PasswordHash),
		CreatedAt:    null.NewTime(usr.CreatedAt.UTC(), !usr.CreatedAt.IsZero()),
		UpdatedAt:    null.NewTime(usr.UpdatedAt.UTC(), !usr.UpdatedAt.IsZero()),
		LastLogin:    null.NewTime(usr.LastLogin.UTC(), !usr.LastLogin.IsZero()),
	}
	if usr.ID != "" {
		u.ID = usr.ID
	}
	return u
}

func (repo userRepository) unboil(usr *models.User) user.User {
	if usr == nil {
		return user.User{}
	}
	return user.User{
		ID:           usr.ID,
		Name:         usr.Name.String,
		Username:     usr.Username.String,
		Email:        usr.Email.String,
		IsActive:     usr.IsActive.Ptr(),
		Roles:        usr.Roles,
		PasswordHash: usr.PasswordHash.Bytes,
		CreatedAt:    usr.CreatedAt.Time,
		UpdatedAt:    usr.UpdatedAt.Time,
		LastLogin:    usr.LastLogin.Time,
	}
}

func (repo userRepository) unboilSlice(slice models.UserSlice) []user.User {
	users := make([]user.User, 0, len(slice))
	for _, u := range slice {
		users = append(users, repo.unboil(u))
	}
	return users
}

// trapNoRowsErr maps psql "no rows" err to user.ErrNotFound
func (repo userRepository) trapNoRowsErr(err error, msg string) error {
	if err == sql.ErrNoRows {
		return user.ErrNotFound
	}
	return errors.Wrap(err, msg)
}

func (repo userRepository) CheckUsernameUniqueness(ctx context.Context, username, email string, excludedUsers []user.User, exec ...core.DBExecutor) error {
	mods := []qm.QueryMod{
		qm.Expr(qm.Where(fmt.Sprintf("%s = ? OR %s = ?", models.UserColumns.Username, models.UserColumns.Email), username, email)),
	}
	if len(excludedUsers) > 0 {
		ids := make([]string, 0, len(excludedUsers))
		for _, u := range excludedUsers {
			ids = append(ids, u.ID)
		}
		mods = append(mods, models.UserWhere.ID.NIN(ids))
	}

	exists, err := models.Users(mods...).Exists(ctx, repo.getExec(exec))
	if err != nil {
		return errors.Wrap(err, "checking user uniqueness")
	}
	if exists {
		return user.ErrUserExists
	}
	return nil
}

func (repo userRepository) CreateUser(ctx context.Context, usr user.User, exec ...core.DBExecutor) (user.User, error) {
	usr.ID = uuid.New().String()
	u := repo.boil(usr)
	if err := u.Insert(ctx, repo.getExec(exec), boil.Infer()); err != nil {
		return user.User{}, errors.Wrap(err, "inserting user")
	}
	return repo.unboil(u), nil
}

func (repo userRepository) QueryUsers(ctx context.Context, filter *user.QueryFilter, ordering []core.DBOrdering, exec ...core.DBExecutor) ([]user.User, error) {
	var mods []qm.QueryMod

	if filter != nil {
		// users with Name, Username or Email matching the search keyword
		if filter.Search != "" {
			val := "%" + filter.Search + "%"
			mods = append(mods, qm.Expr(qm.Where(
				fmt.Sprintf(
					"%s ILIKE ? OR %s ILIKE ? OR %s ILIKE ?",
					models.UserColumns.Name, models.UserColumns.Username, models.UserColumns.Email),
				val, val, val)))
		}
		// users with any role that starts with any of the provided roles
		if len(filter.Roles) > 0 {
			roleMods := make([]qm.QueryMod, 0, len(filter.Roles))
			for _, role := range filter.Roles {
				roleMods = append(roleMods, qm.Or2(qm.Where(
					fmt.Sprintf(
						"%s IN (SELECT %s FROM \"%s\", UNNEST(%s) user_role WHERE user_role ILIKE ?)",
						models.UserColumns.ID, models.UserColumns.ID, models.TableNames.User, models.UserColumns.Roles), role+"%")))
			}
			mods = append(mods, qm.Expr(roleMods...))
		}
		if filter.IsActive != nil {
			mods = append(mods, models.UserWhere.IsActive.EQ(null.BoolFromPtr(filter.IsActive)))
		}
		if !filter.CreatedFrom.IsZero() {
			mods = append(mods, models.UserWhere.CreatedAt.GTE(null.TimeFrom(filter.CreatedFrom.UTC())))
		}
		if !filter.CreatedTo.IsZero() {
			mods = append(mods, models.UserWhere.CreatedAt.LTE(null.TimeFrom(filter.CreatedTo.UTC())))
		}
	}

	if ordering != nil {
		orderList := make([]string, 0, len(ordering))
		for _, ord := range ordering {
			orderList = append(orderList, ord.String())
		}
		mods = append(mods, qm.OrderBy(strings.Join(orderList, ", ")))
	}

	users, err := models.Users(mods...).All(ctx, repo.getExec(exec))
	if err != nil {
		return nil, errors.Wrap(err, "querying users")
	}
	return repo.unboilSlice(users), nil
}

func (repo userRepository) GetUser(ctx context.Context, filter user.GetFilter, exec ...core.DBExecutor) (user.User, error) {
	var usr *models.User
	var err error
	exe := repo.getExec(exec)

	if filter.ID != "" {
		if _, err = uuid.Parse(filter.ID); err != nil {
			return user.User{}, user.ErrNotFound
		}
		usr, err = models.FindUser(ctx, exe, filter.ID)
		if err != nil {
			return user.User{}, repo.trapNoRowsErr(err, "finding user by ID")
		}
	} else {
		var mod qm.QueryMod

		if filter.Username != "" {
			mod = models.UserWhere.Username.EQ(null.StringFrom(filter.Username))
		} else if filter.Email != "" {
			mod = models.UserWhere.Email.EQ(null.StringFrom(filter.Email))
		} else if filter.UsernameOrEmail != nil {
			var email string
			uname := filter.UsernameOrEmail[0]
			if len(filter.UsernameOrEmail) == 2 {
				email = filter.UsernameOrEmail[1]
			}
			if email == "" {
				email = uname
			} else if uname == "" {
				uname = email
			}
			if email != "" && uname != "" {
				mod = qm.Where(fmt.Sprintf("%s = ? OR %s = ?", models.UserColumns.Username, models.UserColumns.Email), uname, email)
			}
		}

		usr, err = models.Users(mod).One(ctx, exe)
		if err != nil {
			return user.User{}, repo.trapNoRowsErr(err, "finding user")
		}
	}

	return repo.unboil(usr), nil
}

func (repo userRepository) UpdateUser(ctx context.Context, usr user.User, exec ...core.DBExecutor) (user.User, error) {
	u := repo.boil(usr)
	if _, err := u.Update(ctx, repo.getExec(exec), boil.Infer()); err != nil {
		return user.User{}, errors.Wrap(err, "updating user")
	}
	return repo.unboil(u), nil
}

func (repo userRepository) UpdateOrCreateUser(ctx context.Context, usr user.User, exec ...core.DBExecutor) (user.User, error) {
	if usr.ID == "" {
		return repo.CreateUser(ctx, usr, exec...)
	}
	return repo.UpdateUser(ctx, usr, exec...)
}

func (repo userRepository) DeleteUsersByID(ctx context.Context, ids []string, exec ...core.DBExecutor) (int, error) {
	cnt, err := models.Users(models.UserWhere.ID.IN(ids)).DeleteAll(ctx, repo.getExec(exec))
	if err != nil {
		return 0, errors.Wrap(err, "deleting users")
	}
	return int(cnt), nil
}
