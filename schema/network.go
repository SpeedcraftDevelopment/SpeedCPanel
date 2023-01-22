package schema

type Network struct {
	OwnedByTeam       bool   `bson:"team"`
	Owner             string `bson:"owner"`
	Name              string `bson:"name"`
	Created           int64  `bson:"created_timestasmp"`
	DockerID          string `bson:"docker"`
	SpecialContainers struct {
		Traefik string `bson:"traefik"`
		NFS     string `bson:"nfs"`
	} `bson:"special,inline"`
	Containers []int `bson:"containers"`
}
