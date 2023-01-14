package schema

type User struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
	Email    string `bson:"email"`
	Joined   int64  `bson:"joined_timestamp"`
	Plan     string `bson:"subscription"`
	Networks []int  `bson:"networks"`
	InTeams  []int  `bson:"teams"`
}
