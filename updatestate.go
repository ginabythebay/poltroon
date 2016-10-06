package poltroon

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// UpdateState tracks the state of the update process.  Can be
// consulted to know when we are done, and to get an idea of the
// progress
type UpdateState struct {
	attempting int // total packages to attempt

	waitGroup sync.WaitGroup

	mu        sync.Mutex      // protects this group
	beingMade map[string]bool // what we are currently making
	remaining int             // number of packages left to process
}

// NewUpdateState returns a new UpdateState.
func NewUpdateState(pkgCnt int) *UpdateState {
	result := &UpdateState{
		attempting: pkgCnt,
		beingMade:  make(map[string]bool),
		remaining:  pkgCnt,
	}
	result.waitGroup.Add(pkgCnt)
	return result
}

// StartMaking records the fact that we are now making the named package.
func (u *UpdateState) StartMake(name string) {
	u.mu.Lock()
	u.beingMade[name] = true
	u.mu.Unlock()
}

// Finished records the fact that we are now done with the named package.
func (u *UpdateState) Finished(name string) {
	u.mu.Lock()
	delete(u.beingMade, name)
	u.remaining--
	u.mu.Unlock()
	u.waitGroup.Done()
}

// Wait blocks until all packages are done
func (u *UpdateState) Wait() {
	u.waitGroup.Wait()
}

func (u *UpdateState) String() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	making := make([]string, 0, len(u.beingMade))
	for key := range u.beingMade {
		making = append(making, key)
	}
	// Forcing this into a predictable order will avoid the output
	// changing when nothing changed
	sort.Strings(making)
	currentlyMaking := ""
	if len(making) > 0 {
		currentlyMaking = fmt.Sprintf("  Currently making %s", strings.Join(making, ", "))
	}
	switch {
	case u.remaining == 0:
		return ""
	case u.remaining == 1:
		return fmt.Sprintf("1 package remaining.%s", currentlyMaking)
	default:
		return fmt.Sprintf("%d packages remaining.%s", u.remaining, currentlyMaking)
	}
}
