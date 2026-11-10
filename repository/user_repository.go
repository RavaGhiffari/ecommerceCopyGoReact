package repository

import "goFrameworkFirstTry/data"

type UserRepository struct {
	Users []data.User
}

func NewUserRepository(users []data.User) *UserRepository {
	return &UserRepository{
		Users: users,
	}
}

func (repo *UserRepository) GetAll() []data.User {
	return repo.Users
}

func (repo *UserRepository) GetByID(id int) (data.User, bool) {
	for _, user := range repo.Users {
		if user.ID == id {
			return user, true
		}
	}

	return data.User{}, false
}
