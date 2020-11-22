package inmemdb

import (
	"sort"

	"github.com/trezcool/masomo/backend/core/user"
)

var pkCount int

type userRepository struct {
	db *userTable
}

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
	repo.db.mutex.RLock()
	defer repo.db.mutex.RUnlock()

	exclUsrsLen := len(excludedUsers)
	if exclUsrsLen > 1 {
		sort.Slice(excludedUsers, func(i, j int) bool { return excludedUsers[i].ID < excludedUsers[j].ID })
	}

	for _, usr := range repo.query() {
		if usr.Username == username && !isExcluded(usr, excludedUsers, exclUsrsLen) {
			return user.UsernameExistsErr
		}
		if usr.Email == email && !isExcluded(usr, excludedUsers, exclUsrsLen) {
			return user.EmailExistsErr
		}
	}
	return nil
}

func (repo *userRepository) CreateUser(usr user.User) (user.User, error) {
	repo.db.mutex.Lock()
	defer repo.db.mutex.Unlock()

	pkCount++
	usr.ID = pkCount
	repo.db.table[usr.ID] = &usr
	return usr, nil
}

func (repo *userRepository) QueryAllUsers() ([]user.User, error) {
	repo.db.mutex.RLock()
	defer repo.db.mutex.RUnlock()
	return repo.query(), nil
}

func (repo *userRepository) GetUserByID(id int) (user.User, error) {
	repo.db.mutex.RLock()
	defer repo.db.mutex.RUnlock()

	if usr, ok := repo.db.table[id]; ok {
		return *usr, nil
	}
	return user.User{}, user.NotFoundErr
}

func (repo *userRepository) GetUserByUsername(username string) (user.User, error) {
	repo.db.mutex.RLock()
	defer repo.db.mutex.RUnlock()

	for _, usr := range repo.query() {
		if usr.Username == username {
			return usr, nil
		}
	}
	return user.User{}, user.NotFoundErr
}

func (repo *userRepository) GetUserByEmail(email string) (user.User, error) {
	repo.db.mutex.RLock()
	defer repo.db.mutex.RUnlock()

	for _, usr := range repo.query() {
		if usr.Email == email {
			return usr, nil
		}
	}
	return user.User{}, user.NotFoundErr
}

func (repo *userRepository) GetUserByUsernameOrEmail(username string) (user.User, error) {
	repo.db.mutex.RLock()
	defer repo.db.mutex.RUnlock()

	for _, usr := range repo.query() {
		if (usr.Username == username) || (usr.Email == username) {
			return usr, nil
		}
	}
	return user.User{}, user.NotFoundErr
}

func (repo *userRepository) UpdateUser(usr user.User, isActive *bool) (user.User, error) {
	repo.db.mutex.Lock()
	defer repo.db.mutex.Unlock()

	// only save set fields
	origUsr, ok := repo.db.table[usr.ID]
	if !ok {
		return user.User{}, user.NotFoundErr
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
	repo.db.mutex.Lock()
	defer repo.db.mutex.Unlock()
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
