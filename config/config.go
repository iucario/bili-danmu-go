package config

// Config holds server configuration. Populated from flags or defaults.
type Config struct {
	Host string
	Port int
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Host: "127.0.0.1",
		Port: 12450,
	}
}
