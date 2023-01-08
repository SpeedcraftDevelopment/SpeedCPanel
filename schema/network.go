package schema

type Network struct {
	OwnedByTeam bool   `bson:"team"`
	Owner       int    `bson:"owner"`
	Name        string `bson:"name"`
	DockerID    string `bson:"docker"`
	Containers  []int  `bson:"containers"`
}
