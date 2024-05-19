package smsproxy

import (
	"errors"
	"sync"
)

type repository interface {
	update(id MessageID, newStatus MessageStatus) error
	save(id MessageID) error
	get(id MessageID) (MessageStatus, error)
}

type inMemoryRepository struct {
	db   map[MessageID]MessageStatus
	lock sync.RWMutex
}

func (r *inMemoryRepository) save(id MessageID) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.db[id]; ok {
		return errors.New("message already exists")
	} else {
		r.db[id] = Accepted
	}

	return nil
}

func (r *inMemoryRepository) get(id MessageID) (MessageStatus, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if status, ok := r.db[id]; ok {
		return status, nil
	}

	return NotFound, nil
}

func (r *inMemoryRepository) update(id MessageID, newStatus MessageStatus) error {
	// Set new status for a given message.
	// If message is not in ACCEPTED state already - return an error.
	// If current status is FAILED or DELIVERED - don't update it and return an error. Those are final statuses and cannot be overwritten.
	r.lock.Lock()
	defer r.lock.Unlock()

	var err error = nil

	if status, ok := r.db[id]; ok {
		for _, finalStatus := range finalStatuses {
			if status == finalStatus {
				return errors.New("message is in final state")
			}
		}

		// if status != Accepted && newStatus != Accepted {
		// 	err = errors.New("message is not in ACCEPTED state")
		// }

		r.db[id] = newStatus
	} else {
		err = errors.New("message not found")
	}

	return err
}

func newRepository() repository {
	return &inMemoryRepository{db: make(map[MessageID]MessageStatus), lock: sync.RWMutex{}}
}
