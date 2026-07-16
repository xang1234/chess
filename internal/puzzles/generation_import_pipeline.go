package puzzles

import (
	"context"
	"errors"
	"sync"
)

const orderedGenerationImportQueueCapacity = 1_000

var errOrderedGenerationImportStopped = errors.New("generation import is stopped")

type orderedGenerationCommand struct {
	puzzle    *TrainingPuzzle
	rejection *Rejection
}

// orderedGenerationImport lets CSV parsing run ahead of catalogue persistence
// while keeping every Add and Reject call in input order. The underlying import
// is touched by the writer goroutine only until it exits; lifecycle calls then
// run synchronously after that barrier.
type orderedGenerationImport struct {
	underlying GenerationImport
	commands   chan orderedGenerationCommand
	done       chan struct{}
	failed     chan struct{}

	workerContext context.Context
	cancelWorker  context.CancelFunc

	stateMu  sync.Mutex
	stopping bool
	senders  sync.WaitGroup
	stopOnce sync.Once

	errorMu     sync.Mutex
	writerError error
}

func newOrderedGenerationImport(
	ctx context.Context,
	underlying GenerationImport,
) *orderedGenerationImport {
	workerContext, cancelWorker := context.WithCancel(ctx)
	queued := &orderedGenerationImport{
		underlying:    underlying,
		commands:      make(chan orderedGenerationCommand, orderedGenerationImportQueueCapacity),
		done:          make(chan struct{}),
		failed:        make(chan struct{}),
		workerContext: workerContext,
		cancelWorker:  cancelWorker,
	}
	go queued.run()
	return queued
}

func (i *orderedGenerationImport) run() {
	defer func() {
		i.cancelWorker()
		close(i.done)
	}()
	for command := range i.commands {
		if i.loadWriterError() != nil {
			continue
		}
		if command.puzzle != nil {
			if err := i.underlying.Add(i.workerContext, *command.puzzle); err != nil {
				i.storeWriterError(err)
			}
			continue
		}
		if command.rejection != nil {
			i.underlying.Reject(*command.rejection)
		}
	}
}

func (i *orderedGenerationImport) Add(ctx context.Context, puzzle TrainingPuzzle) error {
	if err := i.loadWriterError(); err != nil {
		return err
	}
	if !i.beginSend() {
		return errOrderedGenerationImportStopped
	}
	defer i.senders.Done()

	command := orderedGenerationCommand{puzzle: &puzzle}
	select {
	case i.commands <- command:
		return i.loadWriterError()
	case <-i.failed:
		return i.loadWriterError()
	case <-ctx.Done():
		return ctx.Err()
	case <-i.workerContext.Done():
		return i.workerContext.Err()
	case <-i.done:
		if err := i.loadWriterError(); err != nil {
			return err
		}
		return errOrderedGenerationImportStopped
	}
}

func (i *orderedGenerationImport) Reject(rejection Rejection) {
	if i.loadWriterError() != nil || !i.beginSend() {
		return
	}
	defer i.senders.Done()

	command := orderedGenerationCommand{rejection: &rejection}
	select {
	case i.commands <- command:
	case <-i.failed:
	case <-i.workerContext.Done():
	case <-i.done:
	}
}

func (i *orderedGenerationImport) Seal(
	ctx context.Context,
	checksum string,
) (ImportReport, error) {
	if err := i.stop(ctx, false); err != nil {
		return ImportReport{}, err
	}
	if err := i.loadWriterError(); err != nil {
		return ImportReport{}, err
	}
	return i.underlying.Seal(ctx, checksum)
}

func (i *orderedGenerationImport) Activate(ctx context.Context) error {
	return i.underlying.Activate(ctx)
}

func (i *orderedGenerationImport) Abandon(ctx context.Context) error {
	stopErr := i.stop(ctx, true)
	if stopErr != nil {
		return errors.Join(i.loadWriterError(), stopErr)
	}
	return errors.Join(i.loadWriterError(), i.underlying.Abandon(ctx))
}

func (i *orderedGenerationImport) stop(ctx context.Context, cancelWorker bool) error {
	if cancelWorker {
		i.cancelWorker()
	}
	i.stopOnce.Do(func() {
		i.stateMu.Lock()
		i.stopping = true
		i.stateMu.Unlock()
		go func() {
			i.senders.Wait()
			close(i.commands)
		}()
	})

	select {
	case <-i.done:
		return nil
	default:
	}
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		i.cancelWorker()
		return ctx.Err()
	}
}

func (i *orderedGenerationImport) beginSend() bool {
	i.stateMu.Lock()
	defer i.stateMu.Unlock()
	if i.stopping {
		return false
	}
	i.senders.Add(1)
	return true
}

func (i *orderedGenerationImport) loadWriterError() error {
	i.errorMu.Lock()
	defer i.errorMu.Unlock()
	return i.writerError
}

func (i *orderedGenerationImport) storeWriterError(err error) {
	if err == nil {
		return
	}
	i.errorMu.Lock()
	defer i.errorMu.Unlock()
	if i.writerError != nil {
		return
	}
	i.writerError = err
	close(i.failed)
}

var _ GenerationImport = (*orderedGenerationImport)(nil)
