package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"
)

type Config struct {
	Server     ServerConfig
	DB         SQLConfig
	Matchmaker MatchmakerConfig
}

type ServerConfig struct {
	Port              string
	DBRequestTimeout  time.Duration
	ServerManagerAddr string
}

type SQLConfig struct {
	DBName string
	DBConn string
}

type MatchmakerConfig struct {
	TeamSize                  int  `json:"teamSize"`
	TeamCount                 int  `json:"teamCount"`
	MaxRatingSpreadToSearch   int  `json:"maxRatingSpreadToSearch"`
	MaxRatingSpreadInGroup    int  `json:"maxRatingSpreadInGroup"`
	CheckReadiness            bool `json:"checkReadiness"`
	SecondsToAcceptMatch      int  `json:"secondsToAcceptMatch"`
	PenaltyForUnacceptedMatch bool `json:"penaltyForUnacceptedMatch"`
	PenaltySeconds            int  `json:"penaltySeconds"`
}

func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:             "8080",
			DBRequestTimeout: time.Duration(2) * time.Second,
		},
		DB: SQLConfig{
			DBName: getEnv("DB", "postgres"),
			DBConn: getEnv("DB_CONN", "postgres://postgres:postgrespw@localhost:32768/goplay?sslmode=disable"),
		},
		Matchmaker: *readMatchmakerConfig(),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func readMatchmakerConfig() *MatchmakerConfig {
	jsonCfg, err := os.Open("../matchmaker_config.json")
	if err != nil {
		log.Printf("Failed to open config file: %s", err)
	}
	defer jsonCfg.Close()

	var cfg MatchmakerConfig
	data, _ := io.ReadAll(jsonCfg)
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("Failed to read config file: %s", err)
	}

	return &cfg
}
