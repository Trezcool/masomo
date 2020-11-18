package in_memdb

import "github.com/trezcool/masomo/backend/apps/user"

var pkCount int

type userRepository struct {
	db *userTable
}

func NewUserRepository(db *DB) user.Repository {
	return &userRepository{db: db.user}
}

func (r *userRepository) query() []user.User {
	res := make([]user.User, 0, len(r.db.t))
	for _, u := range r.db.t {
		res = append(res, *u)
	}
	return res
}

func (r *userRepository) CheckUsernameUniqueness(username, email string) error {
	r.db.mutex.RLock()
	defer r.db.mutex.RUnlock()

	for _, usr := range r.query() {
		if usr.Username == username {
			return user.UsernameExistsErr
		}
		if usr.Email == email {
			return user.EmailExistsErr
		}
	}
	return nil
}

func (r *userRepository) CreateUser(usr user.User) (user.User, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	pkCount++
	usr.ID = pkCount
	r.db.t[pkCount] = &usr
	return usr, nil
}

func (r *userRepository) QueryAllUsers() ([]user.User, error) {
	r.db.mutex.RLock()
	defer r.db.mutex.RUnlock()
	return r.query(), nil
}

func (r *userRepository) GetUserByID(id int) (user.User, error) {
	r.db.mutex.RLock()
	defer r.db.mutex.RUnlock()

	if usr, ok := r.db.t[id]; ok {
		return *usr, nil
	}
	return user.User{}, user.NotFoundErr
}

func (r *userRepository) GetUserByUsername(username string) (user.User, error) {
	r.db.mutex.RLock()
	defer r.db.mutex.RUnlock()

	for _, usr := range r.query() {
		if usr.Username == username {
			return usr, nil
		}
	}
	return user.User{}, user.NotFoundErr
}

func (r *userRepository) GetUserByEmail(email string) (user.User, error) {
	r.db.mutex.RLock()
	defer r.db.mutex.RUnlock()

	for _, usr := range r.query() {
		if usr.Email == email {
			return usr, nil
		}
	}
	return user.User{}, user.NotFoundErr
}

func (r *userRepository) GetUserByUsernameOrEmail(username string) (user.User, error) {
	r.db.mutex.RLock()
	defer r.db.mutex.RUnlock()

	for _, usr := range r.query() {
		if (usr.Username == username) || (usr.Email == username) {
			return usr, nil
		}
	}
	return user.User{}, user.NotFoundErr
}

func (r *userRepository) UpdateUser(usr user.User) (user.User, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()
	return user.User{}, nil
} // TODO

func (r *userRepository) DeleteUser(id int) error { // TODO
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()
	return nil
} // TODO
