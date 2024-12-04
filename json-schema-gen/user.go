package main

//go:generate go run generator.go -type=User $GOFILE

type User struct {
	ID       int    `json:"id" schema:"required,min=1"`
	Name     string `json:"name" schema:"required,minLength=2"`
	Username string `json:"username" schema:"required,minLength=5"`
	Email    string `json:"email" schema:"required,format=email"`
	IsActive bool   `json:"isActive"`

	IgnoreInJson string `json:"-"`
	NoJSONTag    string
}
