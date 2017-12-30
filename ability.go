package astibob

import (
	"regexp"
	"strings"
	"sync"
)

// ability represents an ability as Bob knows it
type ability struct {
	key  string
	isOn bool
	m    sync.Mutex // Locks attributes
	name string
}

// newAbility creates a new ability
func newAbility(name string, isOn bool) *ability {
	return &ability{
		key:  abilityKey(name),
		isOn: isOn,
		name: name,
	}
}

// regexpAbilityKey represents the ability key regexp
var regexpAbilityKey = regexp.MustCompile("[^\\w]+")

// abilityKey creates an ability key
func abilityKey(name string) string {
	return regexpAbilityKey.ReplaceAllString(strings.ToLower(name), "-")
}
