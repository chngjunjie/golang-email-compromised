package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strings"
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

	// Check agains Cache
	result, err := myCache.Get(email.Email)
	if err != nil {
		// error
		c.JSON(http.StatusUnprocessableEntity, "Cache Query Error")
		return
	}

	if result != nil {
		c.JSON(http.StatusCreated, string(result))
		return
	}

	// Check if that any previous email exists in DB
	emailExists := EmailExists(email.Email)

	if emailExists {
		sqlStmt := `SELECT d.domain_name FROM domains as d 
						JOIN compromised_emails as ce
						ON d.email_id = ce.email_id
						WHERE ce.email = $1`
		domains := []Domain{}
		rows, err := Db.Query(sqlStmt, email.Email)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, "Prepare Query Error")
			return
		}

		for rows.Next() {
			s := Domain{}
			if err := rows.Scan(&s.DomainName); err != nil {
				c.JSON(http.StatusUnprocessableEntity, "Prepare Query Error")
				return
			}
			domains = append(domains, s)
		}

		j, err := json.Marshal(domains)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, "Converted Json Error")
			return
		}

		// * Set into Cache
		err = myCache.Set(email.Email, domains, 30*time.Minute)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, "Set Result Cache Error")
			return
		}

		c.JSON(http.StatusCreated, string(j))
	} else {
		var email_id int

		stmt, err := Db.Prepare("INSERT INTO compromised_emails(email, date_added, user_id) VALUES ($1,$2,$3) RETURNING email_id")
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, "Prepare Query Error")
			return
		}
		err = stmt.QueryRow(email.Email, time.Now(), userId).Scan(&email_id)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, "Run Query Error")
			return
		}

		url := "https://haveibeenpwned.com/api/v3/breachedaccount/" + email.Email
		reqest, err := http.NewRequest("GET", url, nil)
		reqest.Header.Add("hibp-api-key", "61e71327fca3400ab0fe3670806828f3")

		if err != nil {
			panic(err)
		}

		resp, err := http.DefaultClient.Do(reqest)

		content, err := ioutil.ReadAll(resp.Body)
		respBody := string(content)

		b := []byte(respBody)
		var ms = []*Domain{}

		err = json.Unmarshal(b, &ms)
		if err != nil {
			panic(err)
		}

		//* Insert into DB
		valueStrings := make([]string, 0, len(ms))
		valueArgs := make([]interface{}, 0, len(ms) * 2)
		i := 0
		for _, row := range ms {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
			valueArgs = append(valueArgs, row.DomainName)
			valueArgs = append(valueArgs, email_id)
			i++
		}

		stm := fmt.Sprintf("INSERT INTO domains (domain_name, email_id) VALUES %s", strings.Join(valueStrings, ","))

		_, err1 := Db.Exec(stm, valueArgs...)
		if err1 != nil {
			c.JSON(http.StatusUnprocessableEntity, "Insert Many into DB Error")
			return
		}

		// * Set into Cache
		err = myCache.Set(email.Email, &ms, 30*time.Minute)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, "Set Result Cache Error")
			return
		}
		c.JSON(http.StatusCreated, respBody)
	}
}

func EmailExists(email string) bool {
	sqlStmt := `SELECT d.domain_name FROM domains as d 
				JOIN compromised_emails as ce
				ON d.email_id = ce.email_id
				WHERE ce.email = $1`
	err := Db.QueryRow(sqlStmt, email).Scan(&email)
	if err != nil {
		if err != sql.ErrNoRows {
			fmt.Println("err ", err)
		}

		return false
	}

	return true
}
