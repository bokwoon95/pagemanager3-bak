package cryptoutil

type pwstore struct {
	metadata *PasswordMetadata
}

type pwstoretx struct {
	store    *pwstore
	metadata *PasswordMetadata
}

func (store *pwstore) GetPasswordMetadata() (*PasswordMetadata, error) {
	return store.metadata, nil
}

func (store *pwstore) SetPasswordMetadata(metadata PasswordMetadata) error {
	store.metadata = &metadata
	return nil
}

func (store *pwstore) BeginTx() (PasswordStoreTx, error) {
	return &pwstoretx{store: store, metadata: store.metadata}, nil
}

func (tx *pwstoretx) GetPasswordMetadata() (*PasswordMetadata, error) {
	return tx.metadata, nil
}

func (tx *pwstoretx) SetPasswordMetadata(metadata PasswordMetadata) error {
	tx.metadata = &metadata
	return nil
}

func (tx *pwstoretx) Commit() error {
	tx.store.metadata = tx.metadata
	return nil
}

func (tx *pwstoretx) Rollback() error {
	tx.metadata = nil
	return nil
}
