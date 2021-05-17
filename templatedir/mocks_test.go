package templatedir

import (
	"sync"
)

type vkey struct{ localeCode, namespace, name string }

type vstore struct {
	mu     *sync.RWMutex
	values map[vkey]NullString
	rows   map[vkey][]map[string]interface{}
}

func newVstore() *vstore {
	return &vstore{
		mu:     &sync.RWMutex{},
		values: make(map[vkey]NullString),
		rows:   make(map[vkey][]map[string]interface{}),
	}
}

func (store *vstore) GetValue(localeCode, namespace, name string) (value NullString, err error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	// fmt.Printf("lookup: localeCode=%s\n, namespace=%s, name=%s", localeCode, namespace, name)
	return store.values[vkey{localeCode: localeCode, namespace: namespace, name: name}], nil
}

func (store *vstore) GetRows(localeCode, namespace, name string) (rows []map[string]interface{}, err error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	// fmt.Printf("lookup: localeCode=%s\n, namespace=%s, name=%s", localeCode, namespace, name)
	return store.rows[vkey{localeCode: localeCode, namespace: namespace, name: name}], nil
}

func (store *vstore) BeginTx() (ValueStoreTx, error) {
	return &vstoretx{
		mu:     &sync.RWMutex{},
		values: make(map[vkey]NullString),
		rows:   make(map[vkey][]map[string]interface{}),
		store:  store,
	}, nil
}

type vstoretx struct {
	mu     *sync.RWMutex
	values map[vkey]NullString
	rows   map[vkey][]map[string]interface{}
	store  *vstore
}

func (tx *vstoretx) SetValue(localeCode, namespace, name string, value string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.values[vkey{localeCode: localeCode, namespace: namespace, name: name}] = NullString{Valid: true, Str: value}
	return nil
}

func (tx *vstoretx) SetRows(localeCode, namespace, name string, rows []map[string]interface{}) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.rows[vkey{localeCode: localeCode, namespace: namespace, name: name}] = rows
	return nil
}

func (tx *vstoretx) Commit() error {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()
	for k, v := range tx.values {
		tx.store.values[k] = v
	}
	for k, v := range tx.rows {
		tx.store.rows[k] = v
	}
	return nil
}

func (tx *vstoretx) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	for k := range tx.values {
		delete(tx.values, k)
	}
	for k := range tx.rows {
		delete(tx.rows, k)
	}
	return nil
}
