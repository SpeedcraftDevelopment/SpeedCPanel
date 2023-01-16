package schema

type Network struct {
	OwnedByTeam bool   `bson:"team"`
	Owner       string `bson:"owner"`
	Name        string `bson:"name"`
	Created     int64  `bson:"created_timestasmp"`
	DockerID    string `bson:"docker"`
	Containers  []int  `bson:"containers"`
}
