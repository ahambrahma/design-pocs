package models

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Seat struct {
	ID     int    `json:"id"`
	Number string `json:"number"`
	UserID *int   `json:"user_id,omitempty"`
}

type Trip struct {
	ID int `json:"id"`
}
