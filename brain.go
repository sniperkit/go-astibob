package astibob

import "sync"

// brain is a brain as Bob knows it
type brain struct {
	a    map[string]*ability
	m    sync.Mutex // Locks a
	name string
}

// newBrain creates a new brain
func newBrain(name string) *brain {
	return &brain{
		a:    make(map[string]*ability),
		name: name,
	}
}

// ability returns a specific ability based on its name.
func (b *brain) ability(name string) (a *ability, ok bool) {
	b.m.Lock()
	defer b.m.Unlock()
	a, ok = b.a[name]
	return
}

// abilities loops through abilities and execute a function on each of them.
// If an error is returned by the function, the loop is stopped.
func (b *brain) abilities(fn func(a *ability) error) (err error) {
	b.m.Lock()
	defer b.m.Unlock()
	for _, a := range b.a {
		if err = fn(a); err != nil {
			return
		}
	}
	return
}
