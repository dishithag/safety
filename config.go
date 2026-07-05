package summarizer

import "os"

// Config holds runtime configuration for the summarizer job.
type Config struct {
	ServiceName string
	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
}

// LoadConfig builds a Config from environment variables with local-dev defaults.
func LoadConfig(serviceName string) *Config {
	return &Config{
		ServiceName: serviceName,
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
