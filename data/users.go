package data

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func GetUsers() []User {
	return []User{
		{
			ID:    1,
			Name:  "Rava",
			Email: "rava@example.com",
		},
		{
			ID:    2,
			Name:  "John",
			Email: "john@example.com",
		},
		{
			ID:    3,
			Name:  "Alex",
			Email: "alex@example.com",
		},
	}
}
