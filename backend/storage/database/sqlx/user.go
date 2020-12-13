package sqlxrepos

import (
	"context"
	"database/sql"

	_ "github.com/jmoiron/sqlx"

	"github.com/trezcool/masomo/core/user"
)

// todo: + Masterminds/squirrel

type userRepository struct {
}

func NewUserRepository() *userRepository {
	return &userRepository{}
}

func (repo userRepository) CheckUsernameUniqueness(ctx context.Context, db *sql.DB, username, email string, excludedUsers ...user.User) error {
	// SELECT COUNT(*) FROM "user" WHERE (username = $1 OR email = $2) AND ("id" NOT IN ($3,$4)) LIMIT 1;
	return nil
}

func (repo userRepository) CreateUser(ctx context.Context, db *sql.DB, usr user.User) (user.User, error) {
	return user.User{}, nil
}

func (repo userRepository) QueryAllUsers(ctx context.Context, db *sql.DB) ([]user.User, error) {
	return nil, nil
}

func (repo userRepository) GetUserByID(ctx context.Context, db *sql.DB, id int) (user.User, error) {
	return user.User{}, nil
}

func (repo userRepository) GetUserByUsername(ctx context.Context, db *sql.DB, username string) (user.User, error) {
	return user.User{}, nil
}

func (repo userRepository) GetUserByEmail(ctx context.Context, db *sql.DB, email string) (user.User, error) {
	return user.User{}, nil
}

func (repo userRepository) GetUserByUsernameOrEmail(ctx context.Context, db *sql.DB, username string) (user.User, error) {
	return user.User{}, nil
}

func (repo userRepository) FilterUsers(ctx context.Context, db *sql.DB, filter user.QueryFilter) ([]user.User, error) {
	return nil, nil
}

func (repo userRepository) UpdateUser(ctx context.Context, db *sql.DB, usr user.User, isActive ...*bool) (user.User, error) {
	return user.User{}, nil
}

func (repo userRepository) DeleteUsersByID(ctx context.Context, db *sql.DB, ids ...int) error {
	return nil
}
