package repository

import (
	"im-system/internal/model"
	"im-system/pkg/db"

	"gorm.io/gorm"
)

type UserRepository struct {
	orm *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{orm: db.GetDB()}
}

func (r *UserRepository) Create(user *model.User) error {
	return r.orm.Create(user).Error
}

func (r *UserRepository) GetByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.orm.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByUsernameOrEmail(identifier string) (*model.User, error) {
	var u model.User
	if err := r.orm.Where("username = ? OR email = ?", identifier, identifier).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
