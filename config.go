package summarizer

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultSummarizerConcurrency = 3
	defaultGenerationTimeout     = 5 * time.Minute
)

// Config holds runtime configuration for the summarizer job.
type Config struct {
	ServiceName       string
	S3Endpoint        string
	S3Bucket          string
	S3AccessKey       string
	S3SecretKey       string
	NarrativeProvider string
	GenAIHubModel     string
	Concurrency       int
	GenerationTimeout time.Duration
}

// LoadConfig builds a Config from environment variables with local-dev defaults.
func LoadConfig(serviceName string) (*Config, error) {
	concurrency, err := positiveIntEnvOr("SUMMARIZER_CONCURRENCY", defaultSummarizerConcurrency)
	if err != nil {
		return nil, err
	}
	generationTimeout, err := positiveDurationEnvOr("SUMMARIZER_GENERATION_TIMEOUT", defaultGenerationTimeout)
	if err != nil {
		return nil, err
	}

	return &Config{
		ServiceName:       serviceName,
		S3Endpoint:        envOr("S3_ENDPOINT", "http://minio:9000"),
		S3Bucket:          envOr("S3_BUCKET", "dev"),
		S3AccessKey:       envOr("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:       envOr("S3_SECRET_KEY", "minioadmin"),
		NarrativeProvider: envOr("NARRATIVE_PROVIDER", ""),
		GenAIHubModel:     envOr("GENAI_HUB_MODEL", ""),
		Concurrency:       concurrency,
		GenerationTimeout: generationTimeout,
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func positiveIntEnvOr(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", key, value)
	}
	return parsed, nil
}

func positiveDurationEnvOr(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, got %q", key, value)
	}
	return parsed, nil
}
