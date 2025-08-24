package state

import (
	"communication_module/command"
	"communication_module/logger"
	"encoding/json"

	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func StructToMap(s interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ModuleState represents the state of the module
type ModuleState struct {
	Status      string // e.g., "IDLE", "ACTIVE", "SAFE"
	LastCommand command.Command
	LastUpdated int64 // Unix timestamp
}

// Example: update state based on a command
func (ms *ModuleState) Update(cmd command.Command) {
	ms.LastCommand = cmd
	ms.LastUpdated = time.Now().Unix()
	// You can add logic to change Status based on cmd.CMD
	logger.Info("Module state updated:", ms)
}

func InspectPanel(ms *ModuleState, ctx context.Context, rdb *redis.Client) {
	// Logic to inspect the panel
	logger.PubModuleQ(ctx, rdb, "Panel inspection started", map[string]interface{}{}, "INFO")
	ms.Status = "ACTIVE"
	logger.PubModuleQ(ctx, rdb, "Taking Photgraph of Panel", map[string]interface{}{}, "INFO")
	time.Sleep(20 * time.Microsecond) // Simulate time taken to take a photo
	logger.PubModuleQ(ctx, rdb, "Photograph taken", map[string]interface{}{}, "INFO")
}

func ProcessCommand(cmd command.Command, ms *ModuleState, ctx context.Context, rdb *redis.Client) {
	// Core of the module's state management aka state machine
	logger.Info("Recieved Command: ", cmd.CMD)

	switch cmd.CMD {
	case "INSPECT_PANEL":
		logger.Info("Inspecting panel")
		InspectPanel(ms, ctx, rdb)

		// Add logic to inspect panel
	case "THRUST":
		logger.Info("Activating thrust...")
		// Add logic to activate thrust
	case "RESUME":
		logger.Info("Resuming operations...")
		// Add logic to resume operations
	default:
		logger.Error("Unknown command:", cmd.CMD)
	}
}
