package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/redis/go-redis/v9"
)

// Handler is a callback for processing each Pub/Sub message.
type Handler func(ctx context.Context, channel, payload string) error

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Info prints informational messages in yellow
func Info(a ...interface{}) {
	fmt.Print("\r\n")
	fmt.Print("\033[33m")
	fmt.Printf("[%s] ", timestamp())
	fmt.Print(a...)
	fmt.Print("\033[0m")
}

// Info prints informational messages in yellow
func Warning(a ...interface{}) {
	fmt.Print("\r\n")
	fmt.Print("\033[38;5;208m")
	fmt.Printf("[%s] ", timestamp())
	fmt.Print(a...)
	fmt.Print("\033[0m")
}

// Error prints error messages in red
func Error(a ...interface{}) {
	fmt.Print("\r\n")
	fmt.Print("\033[31m")
	fmt.Printf("[%s] ", timestamp())
	fmt.Print(a...)
	fmt.Print("\033[0m")
}

// Error prints error messages in red
func Fatal(a ...interface{}) {
	fmt.Print("\r\n")
	fmt.Print("\033[31m")
	fmt.Printf("[%s] ", timestamp())
	fmt.Print(a...)
	fmt.Print("\033[0m")
}

// Plain prints messages without color
func Plain(a ...interface{}) {
	fmt.Print("\r\n")
	fmt.Printf("[%s] -- ", timestamp())
	fmt.Print(a...)
}

func PubModuleQ(
	ctx context.Context,
	rdb *redis.Client,
	message string,
	system_state map[string]interface{},
	channel string,
	payload_input map[string]interface{}) (int64, error) {

	// Let's try logging to redis
	// Publish JSON to channel
	// Example payload

	if channel == "" {
		channel = "MODULE_Q"
	}
	Plain("Publishing to channel:", channel)

	// Example payload
	payload := map[string]interface{}{
		"msg_id": "12345",
		//"action":       "HEALTH_CHECK",
		"system_state": system_state,
		"message":      message,
		"msg_time":     time.Now().Format(time.RFC3339),
		//"body": map[string]interface{}{
		//	"battery": 85,
		//	"temp":    42.5,
		//},
	}

	// Merge payload_input into payload
	// Overwrite or add new keys
	maps.Copy(payload, payload_input)

	// Marshal to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("json marshal error:", err)
		return 0, fmt.Errorf("json marshal: %w", err)
	}

	n, err := rdb.Publish(ctx, channel, data).Result()
	if err != nil {
		Error("redis publish error:", err)
		return n, fmt.Errorf("publish: %w", err)
	}
	Info("published to", channel, "subs:", n)
	return n, nil
}
