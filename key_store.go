package pagemanager

import (
	"database/sql"

	"github.com/bokwoon95/pagemanager/cryptoutil"
	"github.com/bokwoon95/pagemanager/erro"
	"github.com/bokwoon95/pagemanager/sq"
)

func getKeyByID(dialect string, KEYS pm_KEYS, ID string) sq.Query {
	return sq.SQLite.From(KEYS).Where(KEYS.KEY_ID.EqString(ID))
}

func getKeysByStatus(dialect string, KEYS pm_KEYS, status cryptoutil.KeyStatus, limit int) sq.Query {
	return sq.SQLite.From(KEYS).Where(KEYS.STATUS.EqInt(int(status))).Limit(int64(limit))
}

func setKeysByStatus(dialect string, KEYS pm_KEYS, status cryptoutil.KeyStatus, IDs ...string) sq.Query {
	return sq.SQLite.Update(KEYS).Set(KEYS.STATUS.SetInt(int(status))).Where(KEYS.KEY_ID.In(IDs))
}

func addKeys(dialect string, KEYS pm_KEYS, keys []cryptoutil.Key) sq.Query {
	return sq.SQLite.InsertInto(KEYS).Valuesx(func(col *sq.Column) error {
		for _, key := range keys {
			col.SetString(KEYS.KEY_ID, key.ID)
			col.SetString(KEYS.KEY_CIPHERTEXT, string(key.Contents))
			col.SetInt(KEYS.STATUS, int(key.Status))
		}
		return nil
	})
}

func deleteKeys(dialect string, KEYS pm_KEYS, IDs ...string) sq.Query {
	return sq.SQLite.DeleteFrom(KEYS).Where(KEYS.KEY_ID.In(IDs))
}

func keymapper(key *cryptoutil.Key, KEYS pm_KEYS) func(*sq.Row) error {
	return func(row *sq.Row) error {
		key.ID = row.String(KEYS.KEY_ID)
		key.Contents = row.Bytes(KEYS.KEY_CIPHERTEXT)
		key.Status = cryptoutil.KeyStatus(row.Int(KEYS.STATUS))
		return sq.SkipRows
	}
}

func keysmapper(keys *[]cryptoutil.Key, KEYS pm_KEYS) func(*sq.Row) error {
	return func(row *sq.Row) error {
		var key cryptoutil.Key
		_ = keymapper(&key, KEYS)(row)
		return row.Accumulate(func() error {
			*keys = append(*keys, key)
			return nil
		})
	}
}

type keystore struct {
	db      *sql.DB
	dialect string
}

type keystoretx struct {
	tx      *sql.Tx
	dialect string
}

func (store keystore) GetKeyByID(ID string) (*cryptoutil.Key, error) {
	var key cryptoutil.Key
	KEYS := new_KEYS("k")
	rowCount, err := sq.Fetch(store.db, getKeyByID(store.dialect, KEYS, ID), keymapper(&key, KEYS))
	if err != nil {
		return nil, erro.Wrap(err)
	}
	if rowCount == 0 {
		return nil, nil
	}
	return &key, nil
}

func (store keystore) GetKeysByStatus(status cryptoutil.KeyStatus, limit int) ([]cryptoutil.Key, error) {
	var keys []cryptoutil.Key
	KEYS := new_KEYS("k")
	_, err := sq.Fetch(store.db, getKeysByStatus(store.dialect, KEYS, status, limit), keysmapper(&keys, KEYS))
	if err != nil {
		return nil, erro.Wrap(err)
	}
	return keys, nil
}

func (store keystore) BeginTx() (cryptoutil.KeyStoreTx, error) {
	tx, err := store.db.Begin()
	return keystoretx{tx: tx, dialect: store.dialect}, err
}

func (tx keystoretx) GetKeyByID(ID string) (*cryptoutil.Key, error) {
	var key cryptoutil.Key
	KEYS := new_KEYS("k")
	rowCount, err := sq.Fetch(tx.tx, getKeyByID(tx.dialect, KEYS, ID), keymapper(&key, KEYS))
	if err != nil {
		return nil, erro.Wrap(err)
	}
	if rowCount == 0 {
		return nil, nil
	}
	return &key, nil
}

func (tx keystoretx) GetKeysByStatus(status cryptoutil.KeyStatus, limit int) ([]cryptoutil.Key, error) {
	var keys []cryptoutil.Key
	KEYS := new_KEYS("k")
	_, err := sq.Fetch(tx.tx, getKeysByStatus(tx.dialect, KEYS, status, limit), keysmapper(&keys, KEYS))
	if err != nil {
		return nil, erro.Wrap(err)
	}
	return keys, nil
}

func (tx keystoretx) SetStatusForKeys(status cryptoutil.KeyStatus, IDs ...string) error {
	KEYS := new_KEYS("k")
	_, _, err := sq.Exec(tx.tx, setKeysByStatus(tx.dialect, KEYS, status, IDs...), 0)
	return erro.Wrap(err)
}

func (tx keystoretx) AddKeys(keys []cryptoutil.Key) error {
	KEYS := new_KEYS("k")
	_, _, err := sq.Exec(tx.tx, addKeys(tx.dialect, KEYS, keys), 0)
	return erro.Wrap(err)
}

func (tx keystoretx) DeleteKeys(IDs ...string) error {
	KEYS := new_KEYS("k")
	_, _, err := sq.Exec(tx.tx, deleteKeys(tx.dialect, KEYS, IDs...), 0)
	return erro.Wrap(err)
}

func (tx keystoretx) Commit() error { return tx.tx.Commit() }

func (tx keystoretx) Rollback() error { return tx.tx.Rollback() }
