package main

import (
	"communication_module/command"
	"communication_module/logger"
	"communication_module/pubsub"
	"communication_module/state"
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

// Struct to hold module state
var ms state.ModuleState

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
	fmt.Println("Press 'q' and hit enter to quit - or hit CTRL+C.\r\n")

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
	ticker := time.NewTicker(3000 * time.Millisecond)
	defer ticker.Stop()

	// Start heartbeat
	go heartbeat(ctx, rdb, "module:heartbeat", 5000*time.Millisecond)
	// --------- [END TIMERS and HEARTBEAT] ---------

	// --------- [START Pub Sub: Command] ---------
	stop, err := pubsub.SubscribeAsync(ctx, rdb, []string{"CMD_Q"}, 4, 1024, recieveCommand)
	if err != nil {
		log.Fatalf("failed to subscribe: %v", err)
	}
	defer stop()
	// --------- [END Pub Sub: Command] ---------

	// --------- [START For Loop] ---------
	for {
		select {
		case <-quit:
			fmt.Println("\n\033[31m Quitting... \033[0m")
			return
		case <-ticker.C:
			pong, err := rdb.Ping(ctx).Result()
			if err != nil {
				fmt.Println("\nCould not connect to Redis:", err)
				return
			}
			logger.Plain("Redis connected: ", pong, "    ")
			ms_state_repr, err := state.StructToMap(ms)
			if err != nil {
				fmt.Println("Error converting state to struct: ", err)
			}
			logger.PubModuleQ(ctx, rdb, "", ms_state_repr, "MODULE_Q")

		}
	}

}

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

func recieveCommand(ctx context.Context, rdb *redis.Client, channel, payload string) error {
	// Handle the incoming command
	log.Printf("Received command on %s: %s", channel, payload)

	cmd := command.ParseCommand(payload)
	logger.Info("Parsed Command: ", cmd)
	logger.Error("Command Counter: ", cmd.CMD_COUNTER)

	state.ProcessCommand(cmd, &ms, ctx, rdb)
	return nil

}
