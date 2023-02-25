package main

type Config struct {
	Docker struct {
		Host      string
		Version   string
		TLSConfig struct {
			CACertPath string
			CertPath   string
			KeyPath    string
		}
		Headers map[string]string
	}
	DB struct {
		Hostname string
		Port     int
		Database string
	}
	Images  map[string]string
	ApiKeys []string
	Plans   map[string]struct {
		MaxPlayers int
		RCONTime   int
		TTL        int
	}
	AllowedURLs []string
	Traefik     struct {
		ConfigPath string
	}
}
