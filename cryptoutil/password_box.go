package cryptoutil

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"sync"
)

type PasswordMetadata struct {
	PasswordHash []byte
	KeyParams    []byte
}

type PasswordStore interface {
	GetPasswordMetadata() (*PasswordMetadata, error)
	SetPasswordMetadata(PasswordMetadata) error
	BeginTx() (PasswordStoreTx, error)
}

type PasswordStoreTx interface {
	GetPasswordMetadata() (*PasswordMetadata, error)
	SetPasswordMetadata(PasswordMetadata) error
	Commit() error
	Rollback() error
}

type PasswordBox struct {
	mu       *sync.RWMutex
	pwKey    []byte
	pwStore  PasswordStore
	keyStore KeyStore
}

func NewPasswordBox(pwStore PasswordStore, keyStore KeyStore) (*PasswordBox, error) {
	if pwStore == nil {
		return nil, fmt.Errorf("PasswordStore cannot be nil")
	}
	return &PasswordBox{
		mu:       &sync.RWMutex{},
		pwStore:  pwStore,
		keyStore: keyStore,
	}, nil
}

func (box *PasswordBox) CanSetPassword() error {
	metadata, err := box.pwStore.GetPasswordMetadata()
	if err != nil {
		return err
	}
	if metadata != nil {
		return fmt.Errorf("cannot set password because existing password found")
	}
	if box.keyStore != nil {
		keys, err := getAllKeys(box.keyStore)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			return fmt.Errorf("cannot set password because existing keys found")
		}
	}
	return nil
}

func verifyPassword(store interface {
	GetPasswordMetadata() (*PasswordMetadata, error)
}, password []byte) (*PasswordMetadata, error) {
	metadata, err := store.GetPasswordMetadata()
	if err != nil {
		return nil, err
	}
	if metadata == nil {
		return nil, fmt.Errorf("no password found")
	}
	err = CompareHashAndPassword(metadata.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

func setPassword(store interface {
	SetPasswordMetadata(PasswordMetadata) error
}, password []byte) (pwKey []byte, err error) {
	var metadata PasswordMetadata
	metadata.PasswordHash, err = GenerateFromPassword(password, NewParams(nil))
	if err != nil {
		return nil, err
	}
	keyParams := NewParams(nil)
	metadata.KeyParams, err = keyParams.MarshalText()
	if err != nil {
		return nil, err
	}
	pwKey, err = DeriveKey(password, keyParams)
	if err != nil {
		return nil, err
	}
	err = store.SetPasswordMetadata(metadata)
	if err != nil {
		return nil, err
	}
	return pwKey, nil
}

func (box *PasswordBox) SetPassword(password []byte) error {
	if len(password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}
	err := box.CanSetPassword()
	if err != nil {
		return err
	}
	pwKey, err := setPassword(box.pwStore, password)
	if err != nil {
		return err
	}
	box.mu.Lock()
	box.pwKey = pwKey
	box.mu.Unlock()
	return nil
}

func (box *PasswordBox) EnterPassword(password []byte) error {
	metadata, err := verifyPassword(box.pwStore, password)
	if err != nil {
		return err
	}
	var keyParams Params
	err = keyParams.UnmarshalText(metadata.KeyParams)
	if err != nil {
		return err
	}
	pwKey, err := DeriveKey(password, keyParams)
	if err != nil {
		return err
	}
	box.mu.Lock()
	box.pwKey = pwKey
	box.mu.Unlock()
	return nil
}

func (box *PasswordBox) PasswordEntered() bool {
	var exists bool
	box.mu.RLock()
	exists = len(box.pwKey) > 0
	box.mu.RUnlock()
	return exists
}

func (box *PasswordBox) ChangePassword(oldPassword, newPassword []byte) error {
	if len(newPassword) == 0 {
		return fmt.Errorf("password cannot be empty")
	}
	var oldKey, newKey []byte
	var err error
	defer func() {
		if err != nil {
			return
		}
		box.mu.Lock()
		box.pwKey = newKey
		box.mu.Unlock()
	}()
	if box.keyStore == nil {
		_, err = verifyPassword(box.pwStore, oldPassword)
		if err != nil {
			return err
		}
		newKey, err = setPassword(box.pwStore, newPassword)
		return err
	}
	pwTx, err := box.pwStore.BeginTx()
	if err != nil {
		return err
	}
	err = func() (err2 error) {
		defer commitOrRollback(pwTx, &err2)
		var metadata *PasswordMetadata
		metadata, err2 = verifyPassword(pwTx, oldPassword)
		if err2 != nil {
			return err2
		}
		newKey, err2 = setPassword(pwTx, newPassword)
		if err2 != nil {
			return err2
		}
		if box.keyStore == nil {
			return nil
		}
		var oldKeyParams Params
		err2 = oldKeyParams.UnmarshalText(metadata.KeyParams)
		if err2 != nil {
			return err2
		}
		oldKey, err2 = DeriveKey(oldPassword, oldKeyParams)
		if err2 != nil {
			return err2
		}
		var keys []Key
		var IDs []string
		keys, err2 = getAllKeys(box.keyStore)
		if err2 != nil {
			return err2
		}
		for i, key := range keys {
			IDs = append(IDs, key.ID)
			keys[i].Contents, err = decrypt(oldKey, keys[i].Contents)
			if err != nil {
				return err
			}
			keys[i].Contents, err2 = encrypt(newKey, keys[i].Contents)
			if err2 != nil {
				return err2
			}
		}
		keyTx, err2 := box.keyStore.BeginTx()
		if err2 != nil {
			return err2
		}
		err2 = func() (err3 error) {
			defer commitOrRollback(keyTx, &err3)
			err3 = keyTx.DeleteKeys(IDs...)
			if err3 != nil {
				return err3
			}
			err3 = keyTx.AddKeys(keys)
			if err3 != nil {
				return err3
			}
			return nil
		}()
		return err2
	}()
	return err
}

func (box *PasswordBox) getKey() (key []byte, err error) {
	box.mu.RLock()
	pwKey := box.pwKey
	box.mu.RUnlock()
	if len(pwKey) == 0 {
		return nil, fmt.Errorf("password not entered")
	}
	return key, nil
}

func (box *PasswordBox) Encrypt(plaintext []byte) (ciphertext []byte, err error) {
	key, err := box.getKey()
	if err != nil {
		return nil, err
	}
	return encrypt(key, plaintext)
}

func (box *PasswordBox) Decrypt(ciphertext []byte) (plaintext []byte, err error) {
	key, err := box.getKey()
	if err != nil {
		return nil, err
	}
	return decrypt(key, ciphertext)
}

func (box *PasswordBox) HashEncode(msg []byte) (encodedMsg []byte, err error) {
	key, err := box.getKey()
	if err != nil {
		return nil, err
	}
	hash := hashmsg(key, msg)
	// format: <b64_msg>.<b64_hash>
	encodedMsg = append(encodedMsg, base64Encode(msg)...)
	encodedMsg = append(encodedMsg, '.')
	encodedMsg = append(encodedMsg, base64Encode(hash)...)
	return encodedMsg, nil
}

func (box *PasswordBox) HashDecode(encodedMsg []byte) (msg []byte, err error) {
	key, err := box.getKey()
	if err != nil {
		return nil, err
	}
	// format: <b64_msg>.<b64_hash>
	parts := bytes.Split(encodedMsg, []byte{'.'})
	if len(parts) < 2 {
		return nil, ErrInvalidEncodedMsg
	}
	msg, err = base64Decode(parts[0])
	if err != nil {
		return nil, err
	}
	hash, err := base64Decode(parts[1])
	if err != nil {
		return nil, err
	}
	computedHash := hashmsg(key, msg)
	if subtle.ConstantTimeCompare(computedHash, hash) != 1 {
		return nil, ErrInvalidHash
	}
	return msg, nil
}
