package schema

type Team struct {
	Name      string `bson:"name"`
	Users     []int  `bson:"users"`
	Networks  []int  `bson:"networks"`
	Positions []struct {
		Name   string   `bson:"Name"`
		Scopes []string `bson:"scope"`
		Users  []int    `bson:"users,omitempty"`
	} `bson:"inline"`
}
