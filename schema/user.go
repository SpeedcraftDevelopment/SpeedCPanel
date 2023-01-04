package schema

type User struct {
	Username string
	Password string
	Email    string
	Plan     string
	Networks []Network
}
