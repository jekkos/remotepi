package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	MQTTBroker   string
	MQTTClientID string
	MQTTUsername string
	MQTTPassword string
	MarantzPin   int
	LEDPin       int
	OffDelay     time.Duration
	AudioCard    string
}

func Load() *Config {
	cfg := &Config{
		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "remotepi"),
		MQTTUsername: getEnv("MQTT_USERNAME", ""),
		MQTTPassword: getEnv("MQTT_PASSWORD", ""),
		MarantzPin:   getIntEnv("MARANTZ_PIN", 27),
		LEDPin:       getIntEnv("LED_PIN", 17),
		OffDelay:     getDurationEnv("OFF_DELAY", 2*time.Minute),
		AudioCard:    getEnv("AUDIO_CARD", "card2"),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}