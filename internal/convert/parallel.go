package convert

import (
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

// decodeEntriesParallel decodes entries in parallel, preserving input order
// and skipping failures (same semantics as the sequential path).
func decodeEntriesParallel(opts Options, n int, decode func(i int) (image.Image, error)) []image.Image {
	type slot struct {
		img image.Image
		ok  bool
	}
	slots := make([]slot, n)
	var done atomic.Int32
	var progMu sync.Mutex

	forParallel(n, func(i int) {
		img, err := decode(i)
		if err == nil && img != nil {
			slots[i] = slot{img: img, ok: true}
		}
		d := int(done.Add(1))
		progMu.Lock()
		report(opts, "reading", d, n)
		progMu.Unlock()
	})

	frames := make([]image.Image, 0, n)
	for _, s := range slots {
		if s.ok {
			frames = append(frames, s.img)
		}
	}
	return frames
}

// encodeFramesParallel resizes and quantizes frames in parallel, preserving order.
func encodeFramesParallel(opts Options, frames []image.Image, maxWidth int) []*image.Paletted {
	n := len(frames)
	out := make([]*image.Paletted, n)
	var done atomic.Int32
	var progMu sync.Mutex

	forParallel(n, func(i int) {
		resized := resizeMaxWidth(frames[i], maxWidth)
		out[i] = quantize(resized)
		d := int(done.Add(1))
		progMu.Lock()
		report(opts, "encoding", d, n)
		progMu.Unlock()
	})
	return out
}
