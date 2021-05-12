package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"fmt"

	"github.com/google/uuid"
)

type KeyStatus int

const (
	KeyStatusDisabled KeyStatus = 0
	KeyStatusPassive  KeyStatus = 1
	KeyStatusActive   KeyStatus = 2
)

type Key struct {
	ID       string
	Contents []byte
	Status   KeyStatus
}

type KeyStore interface {
	GetKeyByID(ID string) (*Key, error)
	GetKeysByStatus(status KeyStatus, limit int) ([]Key, error)
	BeginTx() (KeyStoreTx, error)
}

type KeyStoreTx interface {
	GetKeysByStatus(status KeyStatus, limit int) ([]Key, error)
	SetStatusForKeys(status KeyStatus, IDs ...string) error
	AddKeys(keys []Key) error
	DeleteKeys(IDs ...string) error
	Commit() error
	Rollback() error
}

func getAllKeys(store interface {
	GetKeysByStatus(status KeyStatus, limit int) ([]Key, error)
}) ([]Key, error) {
	var allKeys []Key
	for _, status := range []KeyStatus{KeyStatusActive, KeyStatusPassive, KeyStatusDisabled} {
		keys, err := store.GetKeysByStatus(status, -1)
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, keys...)
	}
	return allKeys, nil
}

type KeyEncrypter interface {
	Encrypt(plaintext []byte) (ciphertext []byte, err error)
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
}

type KeyBox struct {
	keyStore     KeyStore
	keyEncrypter KeyEncrypter
}

func NewKeyBox(keyStore KeyStore, keyEncrypter KeyEncrypter) (*KeyBox, error) {
	if keyStore == nil {
		return nil, fmt.Errorf("KeyStore cannot be nil")
	}
	if _, ok := keyStore.(staticKey); ok && keyEncrypter != nil {
		keyEncrypter = nil
	}
	return &KeyBox{
		keyStore:     keyStore,
		keyEncrypter: keyEncrypter,
	}, nil
}

func (box *KeyBox) NewKey() (*Key, error) {
	id, _ := uuid.New().MarshalBinary()
	id = base64Encode(id)
	key := &Key{
		ID:       string(id),
		Contents: make([]byte, 32),
		Status:   KeyStatusActive,
	}
	_, err := rand.Read(key.Contents)
	if err != nil {
		return nil, err
	}
	key.Contents = base64Encode(key.Contents)
	if box.keyEncrypter != nil {
		key.Contents, err = box.keyEncrypter.Encrypt(key.Contents)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

func (box *KeyBox) getOrCreateKey() (key *Key, err error) {
	defer func() {
		if key == nil || err != nil {
			return
		}
		if len(key.Contents) == 0 || key.Status != KeyStatusActive {
			key, err = nil, ErrInvalidKey
			return
		}
		if box.keyEncrypter != nil {
			key.Contents, err = box.keyEncrypter.Decrypt(key.Contents)
		}
	}()
	// get key
	keys, err := box.keyStore.GetKeysByStatus(KeyStatusActive, 1)
	if err != nil {
		return nil, err
	}
	if len(keys) > 0 {
		return &keys[0], nil
	}
	// or create key
	key, err = box.NewKey()
	if err != nil {
		return nil, err
	}
	tx, err := box.keyStore.BeginTx()
	if err != nil {
		return nil, err
	}
	err = func() (err2 error) {
		defer commitOrRollback(tx, &err2)
		err2 = tx.AddKeys([]Key{*key})
		if err2 != nil {
			return err2
		}
		keys, err2 = tx.GetKeysByStatus(KeyStatusActive, 1)
		if err != nil {
			return err2
		}
		if len(keys) == 0 {
			return ErrKeyCreationFailed
		}
		key = &keys[0]
		return nil
	}()
	return key, err
}

func (box *KeyBox) getKeyByID(ID string) (*Key, error) {
	key, err := box.keyStore.GetKeyByID(ID)
	if err != nil {
		return nil, err
	}
	if key == nil || key.Status == KeyStatusDisabled {
		return nil, ErrNoKey
	}
	if len(key.Contents) == 0 {
		return nil, ErrInvalidKey
	}
	if box.keyEncrypter != nil {
		key.Contents, err = box.keyEncrypter.Decrypt(key.Contents)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

func (box *KeyBox) Encrypt(plaintext []byte) (ciphertext []byte, err error) {
	key, err := box.getOrCreateKey()
	if err != nil {
		return nil, err
	}
	// format: <key_id>.<raw_ciphertext>
	// format: <raw_ciphertext>
	rawCiphertext, err := encrypt(key.Contents, plaintext)
	if key.ID != "" {
		ciphertext = append(ciphertext, key.ID...)
		ciphertext = append(ciphertext, '.')
	}
	ciphertext = append(ciphertext, rawCiphertext...)
	return ciphertext, nil
}

func (box *KeyBox) Decrypt(ciphertext []byte) (plaintext []byte, err error) {
	var keyID string
	var rawCiphertext []byte
	parts := bytes.Split(ciphertext, []byte{'.'})
	// format: <key_id>.<raw_ciphertext>
	// format: <raw_ciphertext>
	switch len(parts) {
	case 2:
		keyID = string(parts[0])
		rawCiphertext = parts[1]
	case 1:
		rawCiphertext = parts[0]
	default:
		return nil, ErrInvalidCiphertext
	}
	key, err := box.getKeyByID(keyID)
	if err != nil {
		return nil, err
	}
	return decrypt(key.Contents, rawCiphertext)
}

func (box *KeyBox) HashEncode(msg []byte) (encodedMsg []byte, err error) {
	key, err := box.getOrCreateKey()
	if err != nil {
		return nil, err
	}
	hash := hashmsg(key.Contents, msg)
	// format: <key_id>.<b64_msg>.<b64_hash>
	// format: <b64_msg>.<b64_hash>
	if key.ID != "" {
		encodedMsg = append(encodedMsg, key.ID...)
		encodedMsg = append(encodedMsg, '.')
	}
	encodedMsg = append(encodedMsg, base64Encode(msg)...)
	encodedMsg = append(encodedMsg, '.')
	encodedMsg = append(encodedMsg, base64Encode(hash)...)
	return encodedMsg, nil
}

func (box *KeyBox) HashDecode(encodedMsg []byte) (msg []byte, err error) {
	var keyID string
	var b64Msg, b64Hash []byte
	parts := bytes.Split(encodedMsg, []byte{'.'})
	// format: <key_id>.<b64_msg>.<b64_hash>
	// format: <b64_msg>.<b64_hash>
	switch len(parts) {
	case 3:
		keyID = string(parts[0])
		b64Msg = parts[1]
		b64Hash = parts[2]
	case 2:
		b64Msg = parts[0]
		b64Hash = parts[1]
	default:
		return nil, ErrInvalidEncodedMsg
	}
	msg, err = base64Decode(b64Msg)
	if err != nil {
		return nil, err
	}
	hash, err := base64Decode(b64Hash)
	if err != nil {
		return nil, err
	}
	key, err := box.getKeyByID(keyID)
	if err != nil {
		return nil, err
	}
	computedHash := hashmsg(key.Contents, msg)
	if subtle.ConstantTimeCompare(computedHash, hash) != 1 {
		return nil, ErrInvalidHash
	}
	return msg, nil
}

type staticKey struct {
	b []byte
}

func StaticKey(key []byte) KeyStore {
	return staticKey{b: key}
}

func (key staticKey) GetKeyByID(ID string) (*Key, error) {
	if ID != "" {
		return nil, nil
	}
	return &Key{Contents: key.b, Status: KeyStatusActive}, nil
}

func (key staticKey) GetKeysByStatus(status KeyStatus, limit int) ([]Key, error) {
	if status != KeyStatusActive || limit == 0 {
		return nil, nil
	}
	return []Key{{Contents: key.b, Status: KeyStatusActive}}, nil
}

func (key staticKey) BeginTx() (KeyStoreTx, error) { return nil, ErrUnsupported }
