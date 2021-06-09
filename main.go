package main

import (
	"database/sql"

	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	// "github.com/go-redis/cache/v8"
    "time"
	"encoding/json"

	_ "github.com/lib/pq" // here
	"log"
	"net/http"
)

var (
	router = gin.Default()
)

var (
	ctx    = context.Background()
	myCache CacheItf

	// client *redis.Client
	// ring *redis.Ring
	// mycache *cache.Cache
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
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		}),
	}
}

const (
	host     = "localhost"
	port     = 5432
	dbUser   = "postgres"
	password = "admin"
	dbname   = "users_cyber"
)

var (
	Db *sql.DB
)

// type Object struct {
//     Str string
//     Num int
// }

func initDB() {
	var err error

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, dbUser, password, dbname)
	Db, err = sql.Open("postgres", psqlInfo)

	if err != nil {
		panic(err)
	}

	err = Db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")

	// sqlStatement := `
	// 	INSERT INTO accounts (username, password)
	// 	VALUES ('jon@calhoun1.io', 'Jo1nathan')`
	// _, err = Db.Exec(sqlStatement)
	// if err != nil {
	// 	fmt.Println(" err ", err)
	// }
}

func main() {
	initDB()
	initCache()

	// key := "mykey"
    // obj := &Object{
    //     Str: "mystring",
    //     Num: 42,
    // }
	// myCache.Set(key, obj, time.Minute)

	// result, err1 := myCache.Get(key)
	// if err1 != nil {
	// 	fmt.Println("err1 : ", err1)
	// }
	// fmt.Println("result : ", result)
	// deleted, err1 := myCache.Del(key)
	// if err1 != nil {
	// 	fmt.Println("err1 : ", err1)
	// }
	// fmt.Println("deleted : ", deleted)
	// result1, err1 := myCache.Get(key)
	// if err1 != nil {
	// 	fmt.Println("err1 : ", err1)
	// }
	// fmt.Println("result1 : ", result1)
	
	// api
	router.POST("/login", Login)
	router.POST("/todo", TokenAuthMiddleware(), CreateTodo)
	router.POST("/logout", TokenAuthMiddleware(), Logout)
	router.POST("/token/refresh", Refresh)
	router.POST("/token/validate", AuthorizedPage)
	router.POST("/signup", CreateAccount)
	router.POST("/check", checkEmail)

	// html
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	//router.LoadHTMLFiles("templates/upload.html")
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Main website",
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
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
