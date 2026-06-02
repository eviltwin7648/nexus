package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubToken          string
	DatabaseURL          string
	Repos                []string
	PollInterval         time.Duration
	OpenAIAPIKey         string
	OpenAIEmbeddingModel string
	OpenAIChatModel      string
	APIPort              string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	token := os.Getenv("GITHUB_TOKEN")
	dbURL, err := getEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	var repos []string
	if reposVal := os.Getenv("GITHUB_REPOS"); reposVal != "" {
		repos = strings.Split(reposVal, ",")
	}

	pollMins, err := getEnvInt("POLL_INTERVAL_MINS", 30)
	if err != nil {
		return nil, err
	}

	openaiKey, err := getEnv("OPENAI_API_KEY")
	if err != nil {
		return nil, err
	}

	openaiEmbeddingModel := getEnvDefault("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small")

	openaiChatModel := getEnvDefault("OPENAI_CHAT_MODEL", "gpt-4o-mini")

	apiPort := getEnvDefault("API_PORT", "8080")
	return &Config{
		GitHubToken:          token,
		DatabaseURL:          dbURL,
		Repos:                repos,
		PollInterval:         time.Duration(pollMins) * time.Minute,
		OpenAIAPIKey:         openaiKey,
		OpenAIEmbeddingModel: openaiEmbeddingModel,
		OpenAIChatModel:      openaiChatModel,
		APIPort:              apiPort,
	}, nil
}

func getEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) (int, error) {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("%s must be int", key)
		}
		return n, nil
	}
	return def, nil
}
