package sources

import (
	"context"
	"fmt"
	"sync"

	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/rs/zerolog"
)

type Registry struct {
	sources         map[string]cancelableSource
	sourcesMutex    sync.Mutex
	activities      map[string]common.Activity
	activitiesMutex sync.Mutex

	activityQueue chan common.Activity
	errorQueue    chan error
	done          chan struct{}

	logger *zerolog.Logger
}

type cancelableSource struct {
	Source
	cancel context.CancelFunc
}

func NewRegistry(logger *zerolog.Logger) *Registry {
	r := &Registry{
		sources:       make(map[string]cancelableSource),
		activities:    make(map[string]common.Activity),
		activityQueue: make(chan common.Activity),
		errorQueue:    make(chan error),
		done:          make(chan struct{}),
		logger:        logger,
	}

	r.startWorkers(1)

	return r
}

func (r *Registry) Add(source Source) error {
	r.sourcesMutex.Lock()
	defer r.sourcesMutex.Unlock()

	if _, exists := r.sources[source.UID()]; exists {
		return fmt.Errorf("source '%s' already exists", source.UID())
	}

	ctx, cancel := context.WithCancel(context.Background())

	go source.Stream(ctx, r.activityQueue, r.errorQueue)

	r.sources[source.UID()] = cancelableSource{
		Source: source,
		cancel: cancel,
	}

	return nil
}

func (r *Registry) Remove(uid string) error {
	r.sourcesMutex.Lock()
	defer r.sourcesMutex.Unlock()

	h, ok := r.sources[uid]
	if !ok {
		return fmt.Errorf("source '%s' not found", uid)
	}

	h.cancel()
	delete(r.sources, uid)

	return nil
}

func (r *Registry) List() ([]Source, error) {
	r.sourcesMutex.Lock()
	defer r.sourcesMutex.Unlock()

	out := make([]Source, 0, len(r.sources))
	for _, s := range r.sources {
		out = append(out, s.Source)
	}
	return out, nil
}

func (r *Registry) startWorkers(nWorkers int) {
	var wg sync.WaitGroup

	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r.logger.Info().Msgf("Worker %d starting\n", workerID)
			for {
				select {
				case act := <-r.activityQueue:
					r.logger.Info().Msgf("[Worker %d] Processing activity %s\n", workerID, act.UID())

					r.activitiesMutex.Lock()
					r.activities[act.UID()] = act
					r.activitiesMutex.Unlock()

				case err := <-r.errorQueue:
					r.logger.Error().Err(err).Msgf("[Worker %d] Error processing activity %v\n", workerID, err)

				case <-r.done:
					r.logger.Info().Msgf("Worker %d shutting down\n", workerID)
					return
				}
			}
		}(i + 1)
	}
}

func (r *Registry) Shutdown() {
	close(r.done)

	r.sourcesMutex.Lock()
	for _, source := range r.sources {
		source.cancel()
	}
	r.sourcesMutex.Unlock()
}
