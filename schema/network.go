package schema

type Network struct {
	Name       string      `bson:"name"`
	Containers []Container `bson:"containers"`
}
