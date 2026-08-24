package convert

import (
	"fmt"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
)

func parallelWorkers(n int) int {
	if n <= 1 {
		return 1
	}
	w := runtime.GOMAXPROCS(0)
	if w < 1 {
		w = 1
	}
	if w > n {
		w = n
	}
	// Cap peak memory when decoding/resizing many large photos at once.
	const maxWorkers = 16
	if w > maxWorkers {
		w = maxWorkers
	}
	return w
}

// forParallel runs fn(i) for i in [0,n) across a worker pool.
func forParallel(n int, fn func(i int)) {
	workers := parallelWorkers(n)
	if workers == 1 {
		for i := 0; i < n; i++ {
			fn(i)
		}
		return
	}
	jobs := make(chan int, n)
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				fn(i)
			}
		}()
	}
	wg.Wait()
}

// decodeEntriesParallel decodes entries in parallel, preserving input order.
// If any decode fails (or the optional context is canceled), it returns an
// error naming the first failure — failures are never silently dropped.
func decodeEntriesParallel(opts Options, n int, decode func(i int) (image.Image, error)) ([]image.Image, error) {
	if n == 0 {
		return nil, nil
	}
	slots := make([]image.Image, n)
	var (
		done     atomic.Int32
		failN    atomic.Int32
		once     sync.Once
		firstErr error
		progMu   sync.Mutex
	)

	forParallel(n, func(i int) {
		if err := checkCtx(opts); err != nil {
			once.Do(func() { firstErr = err })
			return
		}
		img, err := decode(i)
		if err != nil {
			failN.Add(1)
			once.Do(func() { firstErr = err })
			return
		}
		if img == nil {
			failN.Add(1)
			once.Do(func() { firstErr = fmt.Errorf("decoded nil image") })
			return
		}
		slots[i] = img
		d := int(done.Add(1))
		progMu.Lock()
		report(opts, "reading", d, n)
		progMu.Unlock()
	})

	if firstErr != nil {
		if err := checkCtx(opts); err != nil {
			return nil, err
		}
		failed := int(failN.Load())
		if failed < 1 {
			failed = 1
		}
		return nil, fmt.Errorf("could not decode image frames: %d of %d failed: %w", failed, n, firstErr)
	}
	return slots, nil
}

// encodeFramesParallel resizes and quantizes frames in parallel, preserving order.
// Aborts early if opts.Ctx is canceled.
func encodeFramesParallel(opts Options, frames []image.Image, maxWidth int) ([]*image.Paletted, error) {
	n := len(frames)
	out := make([]*image.Paletted, n)
	var (
		done     atomic.Int32
		once     sync.Once
		firstErr error
		progMu   sync.Mutex
	)

	forParallel(n, func(i int) {
		if err := checkCtx(opts); err != nil {
			once.Do(func() { firstErr = err })
			return
		}
		resized := resizeMaxWidth(frames[i], maxWidth)
		out[i] = quantize(resized)
		d := int(done.Add(1))
		progMu.Lock()
		report(opts, "encoding", d, n)
		progMu.Unlock()
	})

	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}
