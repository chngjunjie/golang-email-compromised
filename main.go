package main

import (
	"database/sql"

	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq" // here
	"log"
	"net/http"
)

var (
	router = gin.Default()
)

var (
	ctx    = context.Background()
	client *redis.Client
)

func init() {
	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // use default Addr
		Password: "",               // no password set
		DB:       0,                // use default DB
	})
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
