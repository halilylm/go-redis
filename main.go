package main

import (
	"context"
	"fmt"
	"go-redis/handler"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/mgo.v2"
)

var limiter *redis_rate.Limiter

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	limiter = redis_rate.NewLimiter(rdb)
	e := echo.New()
	e.Logger.SetLevel(log.ERROR)
	e.Use(middleware.Logger())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: -1,
	}))
	e.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte(handler.JWT_KEY),
		Skipper: func(c echo.Context) bool {
			// Skip authentication for signup and login requests
			if c.Path() == "/login" || c.Path() == "/signup" || c.Path() == "/all" || c.Path() == "" {
				return true
			}
			return false
		},
	}))
	e.Use(rateLimiting)
	db, err := mgo.Dial("localhost")
	if err != nil {
		e.Logger.Fatal(err)
	}
	// Create indices
	if err = db.Copy().DB("twitter").C("users").EnsureIndex(mgo.Index{
		Key:    []string{"email"},
		Unique: true,
	}); err != nil {
		log.Fatal(err)
	}
	// Initialize handler
	h := &handler.Handler{DB: db, Rdb: rdb}

	// Routes
	e.POST("/signup", h.SignUp)
	e.POST("/login", h.SignIn)
	e.POST("/follow/:id", h.FollowUser)
	e.POST("/posts", h.NewPost)
	e.GET("/feed", h.FetchPosts)
	e.GET("/users", h.GetUsers)
	e.GET("/all", h.AllPosts)
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello world")
	})

	go func() {
		// Start server
		e.Logger.Fatal(e.Start(":1323"))
	}()
	closingChannel := make(chan os.Signal, 1)
	signal.Notify(closingChannel, os.Interrupt)
	<-closingChannel
	fmt.Println("starting to shut down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		fmt.Println("couldnt shut down the server...")
	}
}

func rateLimiting(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		res, err := limiter.Allow(c.Request().Context(), "project:123", redis_rate.PerMinute(10))
		if err != nil {
			return err
		}
		h := c.Response().Header()
		h.Set("RateLimit-Remaining", strconv.Itoa(res.Remaining))
		log.Printf("Remaining %d", res.Remaining)
		if res.Allowed == 0 {
			// We are rate limited.

			seconds := int(res.RetryAfter / time.Second)
			h.Set("RateLimit-RetryAfter", strconv.Itoa(seconds))

			// Stop processing and return the error.
			return c.JSON(http.StatusTooManyRequests, "Rate limit exceeded")
		}

		// Continue processing as normal.
		return next(c)
	}
}
