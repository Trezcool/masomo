package dummydb

import (
	"sort"
	"strings"

	"github.com/trezcool/masomo/backend/core/user"
)

var pkCount int

type userRepository struct {
	db *userTable
}

var _ user.Repository = (*userRepository)(nil) // interface compliance check

func NewUserRepository(db *DB) user.Repository {
	return &userRepository{db: db.user}
}

func (repo *userRepository) query() []user.User {
	users := make([]user.User, 0, len(repo.db.table))
	for _, u := range repo.db.table {
		users = append(users, *u)
	}
	return users
}

func (repo *userRepository) CheckUsernameUniqueness(username, email string, excludedUsers ...user.User) error {
	repo.db.RLock()
	defer repo.db.RUnlock()

	exclUsrsLen := len(excludedUsers)
	if exclUsrsLen > 1 {
		sort.Slice(excludedUsers, func(i, j int) bool { return excludedUsers[i].ID < excludedUsers[j].ID })
	}

	for _, usr := range repo.query() {
		if usr.Username == username && !isExcluded(usr, excludedUsers, exclUsrsLen) {
			return user.ErrUsernameExists
		}
		if usr.Email == email && !isExcluded(usr, excludedUsers, exclUsrsLen) {
			return user.ErrEmailExists
		}
	}
	return nil
}

func (repo *userRepository) CreateUser(usr user.User) (user.User, error) {
	repo.db.Lock()
	defer repo.db.Unlock()

	pkCount++
	usr.ID = pkCount
	repo.db.table[usr.ID] = &usr
	return usr, nil
}

func (repo *userRepository) QueryAllUsers() ([]user.User, error) {
	repo.db.RLock()
	defer repo.db.RUnlock()
	return repo.query(), nil
}

func (repo *userRepository) GetUserByID(id int) (user.User, error) {
	repo.db.RLock()
	defer repo.db.RUnlock()

	if usr, ok := repo.db.table[id]; ok {
		return *usr, nil
	}
	return user.User{}, user.ErrNotFound
}

func (repo *userRepository) GetUserByUsername(username string) (user.User, error) {
	repo.db.RLock()
	defer repo.db.RUnlock()

	for _, usr := range repo.query() {
		if usr.Username == username {
			return usr, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (repo *userRepository) GetUserByEmail(email string) (user.User, error) {
	repo.db.RLock()
	defer repo.db.RUnlock()

	for _, usr := range repo.query() {
		if usr.Email == email {
			return usr, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (repo *userRepository) GetUserByUsernameOrEmail(username string) (user.User, error) {
	repo.db.RLock()
	defer repo.db.RUnlock()

	for _, usr := range repo.query() {
		if (usr.Username == username) || (usr.Email == username) {
			return usr, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (repo *userRepository) FilterUsers(filter user.QueryFilter) ([]user.User, error) {
	repo.db.RLock()
	defer repo.db.RUnlock()

	users := repo.query()

	// users with search keyword matching any Name, Username or Email ?
	if filter.Search != "" {
		var filtered []user.User
		for _, u := range users {
			if strings.Contains(strings.ToLower(u.Username), strings.ToLower(filter.Search)) ||
				strings.Contains(strings.ToLower(u.Email), strings.ToLower(filter.Search)) ||
				strings.Contains(strings.ToLower(u.Name), strings.ToLower(filter.Search)) {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}
	// users with any of the specified roles
	if users != nil && filter.Roles != nil && len(filter.Roles) > 0 {
		var filtered []user.User
		for _, u := range users {
			for _, r := range filter.Roles {
				if u.RoleStartsWith(r) {
					filtered = append(filtered, u)
					break
				}
			}
		}
		users = filtered
	}
	if users != nil && filter.IsActive != nil {
		var filtered []user.User
		for _, u := range users {
			if u.IsActive == *filter.IsActive {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}
	if users != nil && !filter.CreatedFrom.IsZero() {
		var filtered []user.User
		timeUTC := filter.CreatedFrom.UTC()
		for _, u := range users {
			if u.CreatedAt.Equal(timeUTC) || u.CreatedAt.After(timeUTC) {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}
	if users != nil && !filter.CreatedTo.IsZero() {
		var filtered []user.User
		timeUTC := filter.CreatedTo.UTC()
		for _, u := range users {
			if u.CreatedAt.Before(timeUTC) || u.CreatedAt.Equal(timeUTC) {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}

	return users, nil
}

func (repo *userRepository) UpdateUser(usr user.User, isActive *bool) (user.User, error) {
	repo.db.Lock()
	defer repo.db.Unlock()

	// only save set fields
	origUsr, ok := repo.db.table[usr.ID]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	if usr.Roles != nil {
		origUsr.Roles = usr.Roles
	}
	if usr.PasswordHash != nil {
		origUsr.PasswordHash = usr.PasswordHash
	}
	if isActive != nil {
		origUsr.IsActive = *isActive
	}
	origUsr.Name = usr.Name
	origUsr.Username = usr.Username
	origUsr.Email = usr.Email
	origUsr.UpdatedAt = usr.UpdatedAt

	repo.db.table[usr.ID] = origUsr
	return *origUsr, nil
}

func (repo *userRepository) DeleteUsersByID(ids ...int) error {
	repo.db.Lock()
	defer repo.db.Unlock()
	for _, id := range ids {
		delete(repo.db.table, id)
	}
	return nil
}

func isExcluded(usr user.User, excludedUsers []user.User, n int) bool {
	if n <= 0 {
		return false
	}
	idx := sort.Search(n, func(i int) bool { return excludedUsers[i].ID >= usr.ID })
	return excludedUsers[idx].ID == usr.ID
}
