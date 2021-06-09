package main

import (
	"database/sql"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/twinj/uuid"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	// "encoding/binary"
)

var user = User{
	ID:       1,
	Username: "username",
	Password: "password",
}

func Logout(c *gin.Context) {
	au, err := ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	deleted, delErr := DeleteAuth(au.AccessUuid)
	if delErr != nil || deleted == 0 {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	c.JSON(http.StatusOK, "Sucessfully logged out")
}

func Login(c *gin.Context) {
	var u User
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "Invalud json provided")
		return
	}

	var err error

	// Get the existing entry present in the database for the given username
	result := Db.QueryRow("select user_id, password from accounts where username=$1", u.Username)
	if err != nil {
		// If there is an issue with the database, return a 500 error
		c.JSON(http.StatusUnauthorized, "Invalid Username Or Password")
		return
	}

	// We create another instance of `Credentials` to store the credentials we get from the database
	storedUsers := &User{}
	// Store the obtained password in `storedCreds`
	err = result.Scan(&storedUsers.ID ,&storedUsers.Password)
	if err != nil {
		// If an entry with the username does not exist, send an "Unauthorized"(401) status
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, "Invalid Username Or Password")
			return
		}

		// If the error is of any other type, send a 500 status
		c.JSON(http.StatusInternalServerError, "Server Error")
		return
	}

	// Compare the stored hashed password, with the hashed version of the password that was received
	if err = bcrypt.CompareHashAndPassword([]byte(storedUsers.Password), []byte(u.Password)); err != nil {
		// If the two passwords don't match, return a 401 status
		c.JSON(http.StatusUnauthorized, "Invalid Username Or Password")
	}

	ts, err := CreateToken(storedUsers.ID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, err.Error())
		return
	}

	saveErr := CreateAuth(storedUsers.ID, ts)
	if saveErr != nil {
		c.JSON(http.StatusUnprocessableEntity, saveErr.Error())
		return
	}

	tokens := map[string]string{
		"access_token":  ts.AccessToken,
		"refresh_token": ts.RefreshToken,
	}
	c.JSON(http.StatusOK, tokens)
}

func CreateToken(userid uint64) (*TokenDetails, error) {
	td := &TokenDetails{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	td.AccessUuid = uuid.NewV4().String()

	td.RtExpires = time.Now().Add(time.Hour * 24 & 7).Unix()
	td.RefreshUuid = uuid.NewV4().String()

	var err error

	os.Setenv("ACCESS_SECRET", "46RJ3xvpN75a5AAIbLEWbPckcVoLWafkH_Z95QlgPDssiY9qWOLC3g5mB0hQ9ywnWl")
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUuid
	atClaims["user_id"] = userid
	atClaims["exp"] = td.AtExpires
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}

	os.Setenv("REFRESH_SECRET", "640OpbhZFnlYEVTsWo_SErUzkcDyRUqFVzIdHN4ZY4gUQHFAsQzi1ZZNI1TK5KVI8uBZ7Y2")
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUuid
	rtClaims["user_id"] = userid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, err
	}

	return td, nil
}

func CreateAuth(userId uint64, td *TokenDetails) error {

	at := time.Unix(td.AtExpires, 0)
	rt := time.Unix(td.RtExpires, 0)
	now := time.Now()

	errAccess := myCache.Set(td.AccessUuid, strconv.Itoa(int(userId)), at.Sub(now))
	if errAccess != nil {
		return errAccess
	}

	errRefresh := myCache.Set(td.RefreshUuid, strconv.Itoa(int(userId)), rt.Sub(now))
	if errRefresh != nil {
		return errRefresh
	}

	return nil
}

type AccessDetails struct {
	AccessUuid string
	UserID     uint64
}

func ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")

	strArr := strings.Split(bearToken, " ")

	if len(strArr) == 2 {
		return strArr[1]
	}

	return ""
}

func VerifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := ExtractToken(r)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("ACCESS_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

func TokenValid(r *http.Request) error {
	token, err := VerifyToken(r)
	if err != nil {
		return err
	}

	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}

	return nil
}

func ExtractTokenMetadata(r *http.Request) (*AccessDetails, error) {
	token, err := VerifyToken(r)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		accessUuid, ok := claims["access_uuid"].(string)

		if !ok {
			return nil, err
		}
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)

		if err != nil {
			return nil, err
		}

		return &AccessDetails{
			AccessUuid: accessUuid,
			UserID:     userId,
		}, nil
	}
	return nil, err

}

func FetchAuth(authD *AccessDetails) (uint64, error) {
	userid, err := myCache.Get(authD.AccessUuid)
	if err != nil {
		return 0, err
	}
	userId, _ := strconv.ParseUint(strings.Trim(string(userid), "\""), 10, 64)

	return userId, nil
}

func DeleteAuth(givenUuid string) (int64, error) {
	deleted, err := myCache.Del(givenUuid)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func Refresh(c *gin.Context) {
	mapToken := map[string]string{}
	if err := c.ShouldBindJSON(&mapToken); err != nil {
		c.JSON(http.StatusUnprocessableEntity, err.Error())
		return
	}

	refreshToken := mapToken["refresh_token"]

	os.Setenv("REFRESH_SECRET", "640OpbhZFnlYEVTsWo_SErUzkcDyRUqFVzIdHN4ZY4gUQHFAsQzi1ZZNI1TK5KVI8uBZ7Y2")
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("REFRESH_SECRET")), nil
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, "Refresh token expired")
		return
	}

	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		c.JSON(http.StatusUnauthorized, err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		refreshUuid, ok := claims["refresh_uuid"].(string)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, err)
			return
		}

		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, err)
			return
		}

		deleted, delErr := DeleteAuth(refreshUuid)
		if delErr != nil || deleted == 0 {
			c.JSON(http.StatusUnauthorized, "unauthorized")
			return
		}

		ts, createErr := CreateToken(userId)
		if createErr != nil {
			c.JSON(http.StatusForbidden, createErr.Error())
			return
		}

		tokens := map[string]string{
			"access_token":  ts.AccessToken,
			"refresh_token": ts.RefreshToken,
		}

		c.JSON(http.StatusCreated, tokens)
	} else {
		c.JSON(http.StatusUnauthorized, "refresh expired")
	}
}

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := TokenValid(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		c.Next()
	}
}

func AuthorizedPage(c *gin.Context) {
	_, err := ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}

	c.JSON(http.StatusCreated, "success")
}
