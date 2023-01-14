package schema

type Network struct {
	OwnedByTeam bool   `bson:"team"`
	Owner       int    `bson:"owner"`
	Name        string `bson:"name"`
	Created     int64  `bson:"created_timestasmp"`
	DockerID    string `bson:"docker"`
	Containers  []int  `bson:"containers"`
}
