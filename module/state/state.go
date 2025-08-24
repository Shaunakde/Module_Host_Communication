package state

import (
	"communication_module/command"
	"communication_module/logger"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/uuid"

	"context"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

// func StructToMap(s interface{}) (map[string]interface{}, error) {
// 	data, err := json.Marshal(s)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var m map[string]interface{}
// 	if err := json.Unmarshal(data, &m); err != nil {
// 		return nil, err
// 	}
// 	return m, nil
// }

func StructToMap(s interface{}) map[string]interface{} {
	data, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// ModuleState represents the state of the module
type ModuleState struct {
	Status       string // e.g., "IDLE", "ACTIVE", "SAFE"
	LastCommand  command.Command
	LastUpdated  int64   // Unix timestamp
	BatteryLevel int64   // Battery level percentage 0-100
	Temperature  float64 // Temperature in Celsius
	//LastCommandReturn map[string]interface{} // To store the result of the last command
}

// Getters
func (ms *ModuleState) GetStatus() string {
	// Make sure module is initialized
	if ms.Status == "" {
		ms.Status = "IDLE"
	}
	logger.Plain("Module status requested:", ms.Status)
	return ms.Status
}

func (ms *ModuleState) GetandRedisLogStatus(ctx context.Context, rdb *redis.Client) string {
	// Make sure module is initialized
	if ms.Status == "" {
		ms.Status = "IDLE"
	}
	logger.Plain("Module status requested:", ms.Status)
	logger.PubModuleQ(ctx, rdb, "Status requested", StructToMap(ms), "MODULE_Q", map[string]interface{}{})
	return ms.Status
}

// Setters
func (ms *ModuleState) SetStatus(NewStatus string) {
	// Do state machine checks here
	// eg. module can't go from SAFE to ACTIVE directly
	if ms.Status == "SAFE" && NewStatus == "ACTIVE" {
		logger.Warning("Cannot set status to ACTIVE from SAFE. Has to IDLE first.")
	}
	if ms.Status == "SAFE" && NewStatus == "IDLE" {
		logger.Info("Returning from Safe Mode")
		ms.Status = "IDLE"
	}
}

// Generic setter using reflection
func (ms *ModuleState) SetField(field string, value interface{}) error {
	v := reflect.ValueOf(ms).Elem()
	f := v.FieldByName(field)
	if !f.IsValid() {
		return fmt.Errorf("no such field: %s", field)
	}
	if !f.CanSet() {
		return fmt.Errorf("cannot set field: %s", field)
	}
	val := reflect.ValueOf(value)
	if f.Type() != val.Type() {
		return fmt.Errorf("provided value type (%s) didn't match field type (%s)", val.Type(), f.Type())
	}
	f.Set(val)
	return nil
}

// Initialize the module state
func Initialize() *ModuleState {
	logger.Info("Module state Initialized:")
	return &ModuleState{
		Status:      "IDLE",
		LastUpdated: time.Now().Unix(),
		LastCommand: command.Command{},
		//LastCommandReturn: nil,
		BatteryLevel: 100,
		Temperature:  75.0,
	}
}

// update state based on a command
func (ms *ModuleState) Update(cmd command.Command) {
	// TODO: Think a lot about this, should be able to centralize updates to host
	ms.LastCommand = cmd
	ms.LastUpdated = time.Now().Unix()
	logger.Info("Module state updated:", ms)
}

func HealthCheck(ms *ModuleState, ctx context.Context, rdb *redis.Client) {
	logger.Plain("Performing health check...")
	return_payload := []string{}
	return_map := map[string]interface{}{}

	logger.Plain(fmt.Sprintf("Starting Health Check: %s", ms.Status))
	logger.PubModuleQ(ctx, rdb, "Health check in progress", StructToMap(ms), "MODULE_Q", map[string]interface{}{})
	return_payload = append(return_payload, fmt.Sprintf("BatteryLevel = %d", ms.BatteryLevel))
	return_payload = append(return_payload, fmt.Sprintf("Temperature = %f", ms.Temperature))

	return_map["type"] = "RET_VALUE"
	return_map["return_params"] = return_payload

	logger.Plain("Sending output of HEALTH_CHECK to MODULE_Q")
	logger.PubModuleQ(ctx, rdb, "Health check completed", StructToMap(ms), "MODULE_Q", return_map)

	ms.Status = "IDLE"

}

func InspectPanel(ms *ModuleState, ctx context.Context, rdb *redis.Client) {

	return_payload := []string{}
	return_map := map[string]interface{}{}

	// Logic to inspect the panel
	// logger.PubModuleQ(ctx, rdb, "", ms_state_repr, "MODULE_Q")
	//logger.PubModuleQ(ctx, rdb, "Panel inspection started", map[string]interface{}{}, "MODULE_Q")

	//st, _ := StructToMap(ms)
	logger.Plain(fmt.Sprintf("Starting Panel Inspection: %s", ms.Status))

	if ms.Status != "IDLE" {
		logger.Warning(fmt.Sprintf("Cannot inspect panel while module is not IDLE. Current status: %s", ms.Status))
		return
	}

	logger.PubModuleQ(ctx, rdb, "Panel inspection started", StructToMap(ms), "MODULE_Q", map[string]interface{}{})
	ms.Status = "ACTIVE"

	logger.PubModuleQ(ctx, rdb, "Taking Photgraph of Panel", StructToMap(ms), "MODULE_Q", map[string]interface{}{})
	n := rand.Intn(2000-200+1) + 200                // Random time to do this between 200ms and 2s
	time.Sleep(time.Duration(n) * time.Millisecond) // Simulate time taken to take a photo
	return_payload = append(return_payload, "OK")
	return_payload = append(return_payload, "image_captured")
	return_payload = append(return_payload, fmt.Sprintf("uri://%s", uuid.New().String()))

	return_map["type"] = "RET_VALUE"
	return_map["return_params"] = return_payload

	logger.Plain("Sending output of INSPECT_PANEL to MODULE_Q")
	logger.PubModuleQ(ctx, rdb, "Photograph taken", StructToMap(ms), "MODULE_Q", return_map)

	ms.Status = "IDLE"
}

func ProcessCommand(cmd command.Command, ms *ModuleState, ctx context.Context, rdb *redis.Client) {
	// Core of the module's state management aka state machine
	logger.Info("Recieved Command: ", cmd.CMD)

	switch cmd.CMD {
	case "INSPECT_PANEL":
		logger.Info("Inspecting panel")

		// TODO: Think about this pattern if time permits
		//ms.Update(command.Command{CMD: "INSPECT_PANEL_FAILED"})
		//ms.Update(cmd)
		//ms.Status = "ACTIVE"
		InspectPanel(ms, ctx, rdb)

		// Add logic to inspect panel
	case "THRUST":
		logger.Info("Activating thrust...")
		// Add logic to activate thrust
	case "RESUME":
		logger.Info("Resuming operations...")
		// Add logic to resume operations

	case "HEALTH_CHECK":
		logger.Info("Performing health check...")
		HealthCheck(ms, ctx, rdb)

	default:
		logger.Error("Unknown command:", cmd.CMD)
	}
}
