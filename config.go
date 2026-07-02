// Package domain holds analyticsapi-specific entities, interfaces, and config.
// It performs no I/O and imports no delivery framework.
package domain

import "os"

// Config holds runtime configuration for the analyticsapi service. Values are
// sourced from the environment so the same binary runs locally and in k8s.
type Config struct {
	ServiceName string
	// ListenAddr is the host:port the HTTP server binds to.
	ListenAddr string
	// S3Endpoint and S3Bucket point at the object store that holds the LLM
	// narratives. Defaults target the in-cluster minio (see deploy/minio.yaml).
	S3Endpoint string
	S3Bucket   string
	S3AccessKey string
	S3SecretKey string
}

// LoadConfig builds a Config from the environment, applying defaults.
func LoadConfig(serviceName string) *Config {
	return &Config{
		ServiceName: serviceName,
		ListenAddr:  envOr("LISTEN_ADDR", ":8080"),
		S3Endpoint:  envOr("S3_ENDPOINT", "http://minio:9000"),
		S3Bucket:    envOr("S3_BUCKET", "dev"),
		S3AccessKey: envOr("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey: envOr("S3_SECRET_KEY", "minioadmin"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
