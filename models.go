package main

type User struct {
	ID       uint64 `json:"id", db:"user_id"`
	Username string `json:"username", db:"username"`
	Password string `json:"password", db:"password"`
	Email    string `json:"email", db:"email"`
}

type TokenDetails struct {
	AccessToken  string
	RefreshToken string
	AccessUuid   string
	RefreshUuid  string
	AtExpires    int64
	RtExpires    int64
}

type Todo struct {
	UserID uint64 `json:"user_id"`
	Title  string `json:"title"`
}

type EmailCompromised struct {
	Email         string `json:"email", db:"email"`
	CheckDateTime string `json:"checkdatetime", db:"checkdatetime"`
}

type Domain struct {
    DomainName string `json:"Name", db:"domain_name"`
	email_id   uint64 `json:"emai_id", db:"email_id"`
}
