package models

import (
	"github.com/google/uuid"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	ID        bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Uuid      uuid.UUID     `json:"uuid" bson:"uuid"`
	Email     string        `json:"email" bson:"email"`
	Password  string        `json:"password" bson:"password"`
	Token     string        `json:"token" bson:"-"`
	Followers []string      `json:"followers" bson:"followers,omitempty"`
}
