package handler

import (
	"go-redis/models"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (h *Handler) SignUp(c echo.Context) error {
	user := models.User{ID: bson.NewObjectId()}
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if user.Email == "" || user.Password == "" {
		return c.JSON(http.StatusBadRequest, "bad request")
	}
	password := []byte(user.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		log.Fatalln(err)
	}
	user.Password = string(hashedPassword)
	db := h.DB.Clone()
	defer db.Close()
	if err := db.DB("twitter").C("users").Insert(user); err != nil {
		if mgo.IsDup(err) {
			return c.JSON(http.StatusBadRequest, "user with that email already exists")
		}
		return c.JSON(http.StatusInternalServerError, "an error occured")
	}
	return c.JSON(http.StatusCreated, user)
}

func (h *Handler) SignIn(c echo.Context) error {
	u := models.User{}
	if err := c.Bind(&u); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	inputPassword := u.Password
	db := h.DB.Clone()
	defer db.Close()
	// check user exists
	if err := db.DB("twitter").C("users").Find(bson.M{"email": u.Email}).One(&u); err != nil {
		if err == mgo.ErrNotFound {
			return c.JSON(http.StatusBadRequest, "user not found")
		}
		return c.JSON(http.StatusInternalServerError, err)
	}
	// check password matches
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(inputPassword)); err != nil {
		return c.JSON(http.StatusBadRequest, "wrong password")
	}
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = u.ID
	claims["exp"] = time.Now().Add(time.Minute * 45).Unix()
	userToken, err := token.SignedString([]byte(JWT_KEY))
	if err != nil {
		return err
	}
	u.Password = ""
	u.Token = userToken
	return c.JSON(http.StatusOK, u)
}

func (h *Handler) FollowUser(c echo.Context) error {
	userID := userIDFromToken(c)
	id := c.Param("id")
	db := h.DB.Clone()
	defer db.Close()
	if err := db.DB("twitter").C("users").UpdateId(bson.ObjectIdHex(id), bson.M{"$addToSet": bson.M{"followers": userID}}); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, "takipçi eklendi")
}

func userIDFromToken(c echo.Context) string {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	return claims["id"].(string)
}

func (h *Handler) GetUsers(c echo.Context) error {
	db := h.DB.Clone()
	defer db.Close()
	var users []models.User
	if err := db.DB("twitter").C("users").Find(bson.M{}).All(&users); err != nil {
		return c.JSON(http.StatusNotFound, "not found")
	}
	return c.JSON(http.StatusOK, users)
}
