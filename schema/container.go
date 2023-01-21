package schema

type Container struct {
	DockerID string `bson:"docker"`
	Name     string `bson:"name"`
	Hostname string `bson:"host"`
	Created  int64  `bson:"created_timestasmp"`
	Image    string `bson:"image"`
	Owner    string `bson:"owner"`
	Volume   struct {
		Name string `bson:"name"`
		Path string `bson:"path"`
	} `bson:"volume,omitempty"`
	IsOwnerTeam bool `bson:"team"`
}
