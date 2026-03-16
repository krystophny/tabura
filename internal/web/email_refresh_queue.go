package web

import (
	"sort"
	"strings"
	"sync"
)

type emailRefreshQueue struct {
	mu      sync.Mutex
	pending map[int64]map[string]struct{}
}

func newEmailRefreshQueue() *emailRefreshQueue {
	return &emailRefreshQueue{pending: map[int64]map[string]struct{}{}}
}

func (q *emailRefreshQueue) add(accountID int64, ids ...string) {
	if q == nil || accountID == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	accountQueue := q.pending[accountID]
	if accountQueue == nil {
		accountQueue = map[string]struct{}{}
		q.pending[accountID] = accountQueue
	}
	for _, id := range ids {
		clean := strings.TrimSpace(id)
		if clean == "" {
			continue
		}
		accountQueue[clean] = struct{}{}
	}
	if len(accountQueue) == 0 {
		delete(q.pending, accountID)
	}
}

func (q *emailRefreshQueue) list(accountID int64) []string {
	if q == nil || accountID == 0 {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	accountQueue := q.pending[accountID]
	if len(accountQueue) == 0 {
		return nil
	}
	out := make([]string, 0, len(accountQueue))
	for id := range accountQueue {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (q *emailRefreshQueue) remove(accountID int64, ids ...string) {
	if q == nil || accountID == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	accountQueue := q.pending[accountID]
	if len(accountQueue) == 0 {
		return
	}
	for _, id := range ids {
		delete(accountQueue, strings.TrimSpace(id))
	}
	if len(accountQueue) == 0 {
		delete(q.pending, accountID)
	}
}
