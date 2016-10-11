package poltroon

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// UpdateState tracks the state of the update process.  Can be
// consulted to know when we are done, and to get an idea of the
// progress
type UpdateState struct {
	// Makes gets an entry whenever a make starts or finishes.  It gets closed
	// once they all complete.
	Makes chan string

	attempting int // total packages to attempt

	waitGroup sync.WaitGroup

	mu        sync.Mutex           // protects this group
	beingMade map[string]time.Time // what we are currently making and when we started
	finished  int                  // number packages finished
}

// NewUpdateState returns a new UpdateState.
func NewUpdateState(pkgCnt int) *UpdateState {
	result := &UpdateState{
		Makes:      make(chan string),
		attempting: pkgCnt,
		beingMade:  make(map[string]time.Time),
	}
	result.waitGroup.Add(pkgCnt)
	return result
}

// StartMake records the fact that we are now making the named package.
func (u *UpdateState) StartMake(name string) {
	u.mu.Lock()
	u.beingMade[name] = time.Now()
	progress := u.progress()
	u.mu.Unlock()
	u.Makes <- progress + "\r"
}

// Finished records the fact that we are now done with the named package.
func (u *UpdateState) Finished(name string) {
	done := false
	u.mu.Lock()
	start, ok := u.beingMade[name]
	if ok {
		delete(u.beingMade, name)
		u.finished++
		if u.finished == u.attempting {
			done = true
		}
	}
	progress := u.progress()
	u.mu.Unlock()
	if ok {
		duration := time.Since(start)
		u.Makes <- fmt.Sprintf("Made %s in %s\n", name, duration)
	}
	u.Makes <- progress + "\r"
	if done {
		close(u.Makes)
	}
	u.waitGroup.Done()
}

// Wait blocks until all packages are done
func (u *UpdateState) Wait() {
	u.waitGroup.Wait()
}

// Assume the terminal is at least 60 chars wide and that we are
// handling less than 100 packages.
const availableCols = 60 - len("(33/44))")

// Assumes the caller holds a lock.
func (u *UpdateState) progress() string {
	making := make([]string, 0, len(u.beingMade))
	for key := range u.beingMade {
		making = append(making, key)
	}
	// Forcing this into a predictable order will avoid the output
	// changing when nothing changed
	sort.Strings(making)
	currentlyMaking := ""
	if len(making) > 0 {
		currentlyMaking = fmt.Sprintf(" Making %s", strings.Join(making, ", "))
		if len(currentlyMaking) > availableCols {
			currentlyMaking = fmt.Sprintf(" Making %d packages", len(making))
		}
	}
	return fmt.Sprintf("(%d/%d)%s", u.finished, u.attempting, currentlyMaking)
}
