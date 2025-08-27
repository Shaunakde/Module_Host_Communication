package main

import (
	"communication_module/command"
	"communication_module/logger"
	"communication_module/pubsub"
	"communication_module/state"
	"math/rand"
	"os/signal"
	"syscall"

	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LOCALS -------------------------------------------------
var ms state.ModuleState
var lastHeartbeats []string
var unchangedTicks int

//---------------------------------------------------------

//var ms *state.ModuleState
//var ms := state.Initialize()

// startHeartbeat launches a 500ms heartbeat that logs to stdout and Redis.
// It SETs "heartbeat:latest" with a TTL and also PUBLISHes on "heartbeat".
func startHeartbeat(ctx context.Context, rdb *redis.Client, quit <-chan struct{}) {
	const interval = 500 * time.Millisecond
	hb := time.NewTicker(interval)

	go func() {
		defer hb.Stop()
		seq := 0
		for {
			select {
			case <-quit:
				fmt.Println("\n\033[36m[Heartbeat] stopped\033[0m")
				return
			case t := <-hb.C:
				seq++
				ts := t.Format(time.RFC3339Nano)
				payload := fmt.Sprintf(`{"seq":%d,"ts":"%s"}`, seq, ts)

				// Echo to console
				fmt.Printf("\n\033[36m[Heartbeat] #%d %s\033[0m", seq, ts)

				// Update Redis "latest" with a TTL slightly > interval (for liveness checks)
				if err := rdb.Set(ctx, "heartbeat:latest", payload, 2*interval).Err(); err != nil {
					fmt.Printf("\n\033[31m[Heartbeat->Redis SET error] %v\033[0m", err)
				}

				// Publish to a channel for listeners (optional but handy)
				if err := rdb.Publish(ctx, "heartbeat", payload).Err(); err != nil {
					fmt.Printf("\n\033[31m[Heartbeat->Redis PUBLISH error] %v\033[0m", err)
				}

				// (Optional) keep a rolling log in Redis
				// _ = rdb.LPush(ctx, "heartbeat:log", payload).Err()
				// _ = rdb.LTrim(ctx, "heartbeat:log", 0, 199).Err()
			}
		}
	}()
}

func main() {

	// Initialize module state
	ms := state.Initialize()

	// Context for Redis ops
	// --------- [START Redis Connection] ---------
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	// --------- [END Redis Connection] ---------

	// Put terminal in raw mode so single keypresses are delivered immediately
	// --------- [START Handle Terminal] ---------
	//oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	//if err != nil {
	//	fmt.Println("Failed to set raw mode:", err)
	//	return
	//}
	//defer func() {
	//	_ = term.Restore(int(os.Stdin.Fd()), oldState)
	//}()
	fmt.Println("Press 'q' and hit enter to quit - or hit CTRL+C.")

	quit := make(chan struct{})
	var once sync.Once
	safeQuit := func() { once.Do(func() { close(quit) }) }

	// Handle Ctrl+C / SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		safeQuit()
	}()

	// Goroutine to read single bytes from stdin
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			b, err := reader.ReadByte()
			if err != nil {
				// If stdin closes or error occurs, just exit the reader loop.
				safeQuit()
				return
			}
			if b == 'q' || b == 'Q' {
				safeQuit()
				return
			} else
			// echo keystrokes or handle other keys here.
			{
				fmt.Printf("\n\033[33m Input: %q \033[0m", b)
			}
		}
	}()
	// --------- [END Handle Terminal] ---------

	// Timers etc
	// --------- [TIMERS and HEARTBEAT] ---------
	ticker_status := time.NewTicker(1000 * time.Millisecond)
	ticker_heartbeat := time.NewTicker(500 * time.Millisecond)
	defer ticker_status.Stop()
	defer ticker_heartbeat.Stop()

	// Start heartbeat
	////go heartbeat(ctx, rdb, "module:heartbeat", 5000*time.Millisecond)
	// --------- [END TIMERS and HEARTBEAT] ---------

	// --------- [START Pub Sub: Command] ---------
	stop, err := pubsub.SubscribeAsync(ctx, rdb, []string{"CMD_Q"}, 4, 1024, ms, recieveCommand)
	if err != nil {
		log.Fatalf("failed to subscribe: %v", err)
	}
	defer stop()
	// --------- [END Pub Sub: Command] ---------

	// --------- [START Main Loop] ---------
	for {
		select {

		case <-quit:
			fmt.Println("\n\033[31m Quitting... \033[0m")
			return

		case <-ticker_status.C:
			pong, err := rdb.Ping(ctx).Result()
			if err != nil {
				fmt.Println("\nCould not connect to Redis:", err)
				return
			}
			logger.Plain("Redis connected: ", pong, "    ")
			//ms_state_repr, err := state.StructToMap(ms)
			ms_state_repr := state.StructToMap(ms)
			//if err != nil {
			//	fmt.Println("Error converting state to struct: ", err)
			//}
			// Let's also get battery and temperature to vary here at random
			ms.BatteryLevel = ms.BatteryLevel - (rand.Int63n(10) - 5)
			ms.Temperature = ms.Temperature - (rand.Float64()*10.0 - 5.0)
			// Randomly clamp values to be reasonable
			if ms.BatteryLevel < 0 {
				ms.BatteryLevel = 0
			}
			if ms.BatteryLevel > 100 {
				ms.BatteryLevel = 80
			}
			if ms.Temperature < -50 {
				ms.Temperature = -30
			}
			if ms.Temperature > 100 {
				ms.Temperature = 80
			}
			logger.PubModuleQ(ctx, rdb, "STATUS", ms_state_repr, "MODULE_Q", map[string]interface{}{})

		case <-ticker_heartbeat.C:
			// Query last 10 host heartbeats
			logger.Plain("Checking for host heartbeat")
			heartbeats, err := rdb.LRange(ctx, "HOST_HEARTBEAT", 0, 9).Result()
			if err != nil {
				logger.Error("Error querying HOST_HEARTBEAT:", err)
				continue
			}

			// Check if heartbeats have changed
			if len(heartbeats) == len(lastHeartbeats) {
				unchanged := true
				for i := range heartbeats {
					if heartbeats[i] != lastHeartbeats[i] {
						unchanged = false
						break
					}
				}
				if unchanged {
					unchangedTicks++
				} else {
					unchangedTicks = 0
				}
			}

			lastHeartbeats = heartbeats
			// React if unchanged for 3 ticks
			if unchangedTicks > 3 {
				logger.Error(
					fmt.Sprintf("Host heartbeat has not updated for %d ticks!", unchangedTicks),
				)
				// Set System state to FAULT
				ms.Status = "SAFE"
				ms.LastUpdated = time.Now().Unix()
				ms_state_repr := state.StructToMap(ms)
				logger.PubModuleQ(ctx, rdb, "FAULT", ms_state_repr, "MODULE_Q", map[string]interface{}{})

				//new code
				return_payload := []string{}
				return_map := map[string]interface{}{}

				return_map["type"] = "RET_VALUE"
				return_payload = append(return_payload, fmt.Sprintf("Missed %d heartbeats. Taking SAFE mode", unchangedTicks))
				return_map["return_params"] = return_payload
				logger.PubModuleQ(ctx, rdb, "Can not resume panel operations", state.StructToMap(ms), "MODULE_Q", return_map)

			} else if unchangedTicks == 0 {
				logger.Plain("Host heartbeat is healthy.")
				// Was the system in fault?
				if ms.Status == "SAFE" {
					logger.Info("System has recovered from fault.")
					// ms.SetField("Status", "IDLE")   // This is nice, but too much generalization?
					ms.SetStatus("IDLE")
				}
				// Set System state indicate healthy
				ms.LastUpdated = time.Now().Unix()
				//ms_state_repr := state.StructToMap(ms)
			} else if unchangedTicks > 0 && unchangedTicks <= 3 {
				// Warning state
				ms.LastUpdated = time.Now().Unix()
				ms_state_repr := state.StructToMap(ms)
				logger.Info(
					fmt.Sprintf("Host heartbeat unchanged for %d ticks.", unchangedTicks),
				)
				logger.PubModuleQ(ctx, rdb, "WARNING", ms_state_repr, "MODULE_Q", map[string]interface{}{})
			}
		}
	}
	// --------- [END Main Loop] ---------

} // End of func main()

func heartbeat(ctx context.Context, rdb *redis.Client, key string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			err := rdb.Set(ctx, key, t.Format(time.RFC3339), interval*5).Err()
			if err != nil {
				fmt.Println("failed to write heartbeat:", err)
			} else {
				fmt.Println("\r\nheartbeat logged at", t)
			}
		}
	}
}

func recieveCommand(ctx context.Context, rdb *redis.Client, channel, payload string, ms *state.ModuleState) error {
	// Handle the incoming command
	log.Printf("Received command on %s: %s", channel, payload)

	cmd := command.ParseCommand(payload)
	logger.Info("Parsed Command: ", cmd)
	logger.Error("Command Counter: ", cmd.CMD_COUNTER)

	fmt.Print(ms)
	state.ProcessCommand(cmd, ms, ctx, rdb)
	return nil

}
