// Package tenantcache provides a bounded in-process cache whose key contains
// every authorization and tenancy dimension that may change a result.
package tenantcache

import (
	"container/list"
	"sync"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

type ResourceKind uint8

const (
	ResourceUnknown ResourceKind = iota
	ResourceProject
	ResourceEnvironment
	ResourceProjectAssignment
	ResourceEnvironmentAssignment
)

type Key struct {
	Tenant            tenant.OrganizationID
	Kind              ResourceKind
	ResourceID        string
	Subject           tenant.AccountID
	ResourceVersion   tenant.Version
	MembershipVersion tenant.Version
	PolicyRevision    uint64
}

type entry[T any] struct {
	key   Key
	value T
}

type Cache[T any] struct {
	mu      sync.Mutex
	maximum int
	order   *list.List
	entries map[Key]*list.Element
}

func New[T any](maximum int) *Cache[T] {
	if maximum < 1 {
		maximum = 1
	}
	if maximum > 10_000 {
		maximum = 10_000
	}
	return &Cache[T]{maximum: maximum, order: list.New(), entries: make(map[Key]*list.Element, maximum)}
}

func (c *Cache[T]) Get(key Key) (T, bool) {
	var zero T
	if c == nil || !key.valid() {
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.entries[key]
	if !ok {
		return zero, false
	}
	c.order.MoveToFront(element)
	return element.Value.(entry[T]).value, true
}

func (c *Cache[T]) Put(key Key, value T) bool {
	if c == nil || !key.valid() {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.entries[key]; ok {
		element.Value = entry[T]{key: key, value: value}
		c.order.MoveToFront(element)
		return true
	}
	element := c.order.PushFront(entry[T]{key: key, value: value})
	c.entries[key] = element
	if c.order.Len() > c.maximum {
		oldest := c.order.Back()
		delete(c.entries, oldest.Value.(entry[T]).key)
		c.order.Remove(oldest)
	}
	return true
}

func (c *Cache[T]) InvalidateTenant(id tenant.OrganizationID) {
	if c == nil || id == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, element := range c.entries {
		if key.Tenant == id {
			delete(c.entries, key)
			c.order.Remove(element)
		}
	}
}

func (k Key) valid() bool {
	if _, err := tenant.NewOrganizationID(string(k.Tenant)); err != nil {
		return false
	}
	if _, err := tenant.NewAccountID(string(k.Subject)); err != nil {
		return false
	}
	var err error
	switch k.Kind {
	case ResourceProject:
		_, err = tenant.NewProjectID(k.ResourceID)
	case ResourceEnvironment:
		_, err = tenant.NewEnvironmentID(k.ResourceID)
	case ResourceProjectAssignment, ResourceEnvironmentAssignment:
		_, err = tenant.NewAssignmentID(k.ResourceID)
	default:
		return false
	}
	return err == nil && k.ResourceVersion > 0 && k.MembershipVersion > 0 && k.PolicyRevision > 0
}
