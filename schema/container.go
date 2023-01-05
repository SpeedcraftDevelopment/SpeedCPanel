package schema

type Container struct {
	DockerID string `bson:"docker"`
	Name     string `bson:"name"`
	Image    string `bson:"image"`
	Tag      string `bson:"tag"`
}
