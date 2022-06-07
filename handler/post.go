package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"go-redis/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"gopkg.in/mgo.v2/bson"
)

func (h *Handler) NewPost(c echo.Context) error {
	u := models.User{
		ID: bson.ObjectIdHex(userIDFromToken(c)),
	}
	post := models.Post{ID: bson.NewObjectId(), From: u.ID.Hex()}
	if err := c.Bind(&post); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if post.Message == "" || post.To == "" {
		return c.JSON(http.StatusBadRequest, "all fields are required")
	}
	db := h.DB.Clone()
	defer db.Close()
	if err := db.DB("twitter").C("posts").Insert(&post); err != nil {
		return c.JSON(http.StatusInternalServerError, "just an internal error")
	}
	return c.JSON(http.StatusCreated, post)
}

func (h *Handler) FetchPosts(c echo.Context) error {
	userID := userIDFromToken(c)
	key := fmt.Sprintf("tweets_%s", userID)
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	db := h.DB.Clone()
	defer db.Close()
	var posts []models.Post
	values, err := h.Rdb.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Println("mongodb working")
			if err := db.DB("twitter").C("posts").Find(bson.M{"to": userID}).Skip((page - 1) * limit).Limit(limit).All(&posts); err != nil {
				return c.JSON(http.StatusNotFound, "nth here")
			}
			jsoned, err := json.Marshal(posts)
			if err != nil {
				log.Fatalln("an error occured")
			}
			if len(posts) > 0 {
				h.Rdb.Set(context.Background(), key, string(jsoned), time.Minute)
			}
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Println("it is from redis")
		if err := json.Unmarshal([]byte(values), &posts); err != nil {
			log.Fatalln(err)
		}
	}
	return c.JSON(http.StatusOK, posts)
}

func (h *Handler) AllPosts(c echo.Context) error {
	time.Sleep(5 * time.Second)
	var posts []models.Post
	key := "posts"
	ctx := context.TODO()
	val, err := h.Rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Println("mongodb worked")
			db := h.DB.Clone()
			defer db.Close()
			if err := db.DB("twitter").C("posts").Find(bson.M{}).All(&posts); err != nil {
				return c.JSON(http.StatusInternalServerError, "just an error")
			}
			jsoned, err := json.Marshal(posts)
			if err != nil {
				log.Fatalln(err)
			}
			h.Rdb.Set(ctx, key, string(jsoned), time.Minute)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Println("redis worked")
		if err := json.Unmarshal([]byte(val), &posts); err != nil {
			log.Fatalln(err)
		}
	}
	return c.JSON(http.StatusOK, posts)
}
