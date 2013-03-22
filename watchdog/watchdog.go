package watchdog

import (
	"time"
)

type nothing struct{}
type Bark nothing

type Watchdog struct {
	Bark     <-chan Bark
	sendBark chan<- Bark
	pet      chan nothing
}

// Return a new watchdog. If watchdog isn't petted at least every
// timeout, it barks (closes the channel w.Bark).
func New(timeout time.Duration) *Watchdog {
	bark := make(chan Bark)
	w := &Watchdog{
		Bark:     bark,
		sendBark: bark,
		pet:      make(chan nothing, 1),
	}
	go w.watch(timeout)
	return w
}

func (w *Watchdog) Pet() {
	select {
	case w.pet <- nothing{}:
		break
	default:
		break
	}
}

// will exit after barking
func (w *Watchdog) watch(timeout time.Duration) {
	defer close(w.sendBark)
	var timer *time.Timer
outer:
	for {
		timer = time.NewTimer(timeout)
		select {
		case <-w.pet:
			timer.Stop()
			continue
		case <-timer.C:
			break outer
		}
	}
}
