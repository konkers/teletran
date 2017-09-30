package main

import (
	"github.com/asdine/storm"
)

type User struct {
	ID      string `json:"id",storm:"id"`
	Credits int    `json:"credits"`
}

func GetUser(db *storm.DB, ID string) (*User, error) {
	var user User
	err := db.One("ID", ID, &user)
	if err == storm.ErrNotFound {
		user.ID = ID
		user.Credits = 0
		db.Save(&user)
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *User) Update(db *storm.DB) error {
	return db.Update(u)
}
