// Package cryptoutil provides both encryption and hashing.
package cryptoutil

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/nacl/secretbox"
)

const nonceSize = 24

var (
	ErrUnsupported       = errors.New("unsupported operation")
	ErrNoKey             = errors.New("no key found")
	ErrKeyCreationFailed = errors.New("failed to create a new key")
	ErrInvalidKey        = errors.New("invalid key")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrInvalidHash       = errors.New("invalid hash")
	ErrInvalidEncodedMsg = errors.New("invalid encoded msg")
)

func base64Encode(src []byte) []byte {
	buf := make([]byte, base64.RawURLEncoding.EncodedLen(len(src)))
	base64.RawURLEncoding.Encode(buf, src)
	return buf
}

func base64Decode(src []byte) ([]byte, error) {
	dbuf := make([]byte, base64.RawURLEncoding.DecodedLen(len(src)))
	n, err := base64.RawURLEncoding.Decode(dbuf, src)
	return dbuf[:n], err
}

func deriveEncryptionKey(key []byte) (encryptionKey [32]byte) {
	hashed := blake2b.Sum512(key)
	copy(encryptionKey[:], hashed[:32])
	return encryptionKey
}

func deriveHashKey(key []byte) (hashKey []byte) {
	hashKey = make([]byte, 32)
	hashed := blake2b.Sum512(key)
	copy(hashKey, hashed[32:])
	return hashKey
}

func commitOrRollback(tx interface {
	Commit() error
	Rollback() error
}, err *error) {
	if *err != nil {
		err2 := tx.Rollback()
		if err2 != nil {
			*err = fmt.Errorf("%w: %s", *err, err2.Error())
		}
	} else {
		*err = tx.Commit()
	}
}

func encrypt(key []byte, plaintext []byte) (ciphertext []byte, err error) {
	encryptionKey := deriveEncryptionKey(key)
	var nonce [nonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	ciphertext = secretbox.Seal(nonce[:], plaintext, &nonce, &encryptionKey)
	ciphertext = base64Encode(ciphertext)
	return ciphertext, nil
}

func decrypt(key []byte, ciphertext []byte) (plaintext []byte, err error) {
	encryptionKey := deriveEncryptionKey(key)
	ciphertext, err = base64Decode(ciphertext)
	if err != nil {
		return nil, err
	}
	var nonce [nonceSize]byte
	copy(nonce[:], ciphertext[:nonceSize])
	plaintext, ok := secretbox.Open(nil, ciphertext[nonceSize:], &nonce, &encryptionKey)
	if !ok {
		return nil, ErrInvalidCiphertext
	}
	return plaintext, nil
}

func hashmsg(key []byte, msg []byte) (hash []byte) {
	hashKey := deriveHashKey(key)
	h, _ := blake2b.New512(hashKey)
	h.Reset()
	h.Write(msg)
	return h.Sum(nil)
}
