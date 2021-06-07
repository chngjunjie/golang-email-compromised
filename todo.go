package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // here
	"golang.org/x/crypto/bcrypt"
)

func CreateTodo(c *gin.Context) {
	var td *Todo
	if err := c.ShouldBindJSON(&td); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "invalid json")
		return
	}

	tokenAuth, err := ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}

	userId, err := FetchAuth(tokenAuth)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}

	td.UserID = userId

	c.JSON(http.StatusCreated, td)
}

func CreateAccount(c *gin.Context) {
	var u User
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "invalid json")
		return
	}

	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which we arbitrarily set as 8 (this value can be more or less, depending on the computing power you wish to utilize)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), 8)
	fmt.Println(" hashedPassword ", hashedPassword)

	// Next, insert the username, along with the hashed password into the database
	if _, err = Db.Query("INSERT INTO accounts(username, password, email) VALUES ($1,$2,$3)", u.Username, string(hashedPassword), u.Email); err != nil {
		// If there is any issue with inserting into the database, return a 500 error
		c.JSON(http.StatusUnprocessableEntity, "Insert DB Error")
		fmt.Println(" err ", err)
		return
	}

	c.JSON(http.StatusCreated, "Insert Successfully")
}

func checkEmail(c *gin.Context) {
	var email EmailCompromised
	if err := c.ShouldBindJSON(&email); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "invalid json")
		return
	}

	au, err := ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}

	// Next, insert the username, along with the hashed password into the database
	var email_id int

	stmt, err := Db.Prepare("INSERT INTO compromised_emails(email, date_added, user_id) VALUES ($1,$2,$3) RETURNING email_id")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, "Prepare Query Error")
		return
	}
	fmt.Println("stmt : ", stmt)

	err = stmt.QueryRow(email.Email, time.Now(), au.UserID).Scan(&email_id)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, "Run Query Error")
		return
	}

	fmt.Println(" email ", email.Email)

	url := "https://haveibeenpwned.com/api/v3/breachedaccount/" + email.Email
	reqest, err := http.NewRequest("GET", url, nil)
	reqest.Header.Add("hibp-api-key", "61e71327fca3400ab0fe3670806828f3")

	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(reqest)
	fmt.Println(" resp ", resp)

	content, err := ioutil.ReadAll(resp.Body)
	respBody := string(content)
	fmt.Println(" respBody ", respBody)
	xt := reflect.TypeOf(respBody).Kind()
	fmt.Printf("%T: %s\n", xt, xt)

	b := []byte(respBody)
	fmt.Println(b)

	var m1 interface{}

	json.Unmarshal(b, &m1)
	switch v := m1.(type) {
	case []interface{}:
		for _, x := range v {
			fmt.Println("this is b", x.(map[string]interface{})["Name"])
		}
	default:
		fmt.Println("No type found")
	}

	var ms = []*Domain{}

	err = json.Unmarshal(b, &ms)
	if err != nil {
		panic(err)
	}

	for _, row := range ms {
		if _, err = Db.Query("INSERT INTO domains(domain_name, email_id) VALUES ($1,$2)", row.DomainName, email_id); err != nil {
			// If there is any issue with inserting into the database, return a 500 error
			c.JSON(http.StatusUnprocessableEntity, "Insert DB Error")
			fmt.Println(" err ", err)
			return
		}
	}

	c.JSON(http.StatusCreated, respBody)
}
