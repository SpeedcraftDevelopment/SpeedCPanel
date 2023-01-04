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
		Engine    string
		Path      string
		Options   map[string]string
		Namespace string
	}
}
