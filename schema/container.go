package schema

type Container struct {
	DockerID string `bson:"docker"`
	Name     string `bson:"name"`
	Hostname string `bson:"host"`
	Image    string `bson:"image"`
}
