package pubsub

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Handler is a callback for processing each Pub/Sub message.
type Handler func(ctx context.Context, rdb *redis.Client, channel, payload string) error

// SubscribeAsync subscribes to Redis channels and dispatches messages to a worker pool.
func SubscribeAsync(ctx context.Context, rdb *redis.Client, channels []string, workers, buf int, h Handler) (stop func(), err error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if buf <= 0 {
		buf = 1024
	}

	ps := rdb.Subscribe(ctx, channels...)
	if _, err := ps.Receive(ctx); err != nil {
		return nil, err
	}

	msgCh := ps.Channel(redis.WithChannelSize(buf))

	wg := &sync.WaitGroup{}
	wg.Add(workers)

	workerCtx, cancel := context.WithCancel(ctx)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-workerCtx.Done():
					return
				case m, ok := <-msgCh:
					if !ok {
						return
					}
					callCtx, cancel := context.WithTimeout(workerCtx, 30*time.Second)
					if err := h(callCtx, rdb, m.Channel, m.Payload); err != nil {
						log.Printf("[worker %d] handler error: %v (channel=%s)", id, err, m.Channel)
					}

					cancel()
				}
			}
		}(i + 1)
	}

	done := make(chan struct{})
	go func() {
		<-workerCtx.Done()
		_ = ps.Close()
		close(done)
	}()

	stop = func() {
		cancel()
		wg.Wait()
		<-done
	}

	return stop, nil
}
