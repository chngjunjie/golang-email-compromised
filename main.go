package main

import (
	"database/sql"

	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
    "time"
	"encoding/json"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // here
	"log"
	"net/http"
	"os"
)

var (
	router = gin.Default()
)

var (
	ctx    = context.Background()
	myCache CacheItf
)

type CacheItf interface {
	Set(key string, data interface{}, expiration time.Duration) error
	Get(key string) ([]byte, error)
	Del(key string) (int64, error)
}

type RedisCache struct {
	client *redis.Client
}
func (r *RedisCache) Set(key string, data interface{}, expiration time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, b, expiration).Err()
}

func (r *RedisCache) Get(key string) ([]byte, error) {
	result, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}

	return result, err
}

func (r *RedisCache) Del(key string) (int64, error) {
	result, err := r.client.Del(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}

	return result, err
}

func initCache() {
	myCache = &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     os.Getenv("RD_HOST") + ":" + os.Getenv("RD_PORT"),
			Password: "", // no password set
			DB:       0,  // use default DB
		}),
	}
}

// Databbase Postgres
var (
	Db *sql.DB
)

func initDB() {

	// url := fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable",
	// 		os.Getenv("DBUSER"),
	// 		os.Getenv("DBPASSWORD"),
	// 		os.Getenv("DBHOST"),
	// 		os.Getenv("DBPORT"),
	// 		os.Getenv("DBNAME"))
	
	// Connect for local usage
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBUSER"), os.Getenv("DBPASSWORD"), os.Getenv("DBNAME"))
	fmt.Println("psqlInfo : ", psqlInfo)


	var err error
	Db, err = sql.Open("postgres", psqlInfo)

	if err != nil {
		panic(err)
	}

	err = Db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")
}

func main() {

	err := godotenv.Load(".env")

	if err != nil {
	  log.Fatalf("Error loading .env file")
	}

	initDB()
	initCache()
	
	// api
	router.POST("/api/login", Login)
	router.POST("/api/logout", TokenAuthMiddleware(), Logout)
	router.POST("/api/token/refresh", Refresh)
	router.POST("/api/token/validate", AuthorizedPage)
	router.POST("/api/signup", CreateAccount)
	router.POST("/api/check", checkEmail)

	// html
	router.LoadHTMLGlob("./nginx/templates/*")
	router.Static("/static", "./nginx/static")

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Main website",
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Main website",
		})
	})

	router.GET("/signup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "signup.html", gin.H{
			"title": "Main website",
		})
	})

	router.GET("/home", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", gin.H{
			"title": "Main website",
		})
	})

	log.Fatal(router.Run(":8080"))
}
