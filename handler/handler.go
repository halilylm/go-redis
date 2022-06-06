package handler

import (
	"github.com/go-redis/redis/v8"
	"gopkg.in/mgo.v2"
)

type Handler struct {
	DB  *mgo.Session
	Rdb *redis.Client
}

const JWT_KEY = "fEhMraol7vGlrbkbcW8pRU4Nyg3dZHfGwGGRwT74TU836MnMryhCCKYoYSyQRT2w"
