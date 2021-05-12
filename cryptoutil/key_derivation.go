package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type KeyParams struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
	Salt    []byte
	err     error
}

func NewKeyParams(salt []byte) KeyParams {
	p := KeyParams{
		Memory:  64 * 1024,
		Time:    1,
		Threads: 4,
		KeyLen:  32,
		Salt:    salt,
	}
	if p.Salt == nil {
		p.Salt = make([]byte, 16)
		_, p.err = rand.Read(p.Salt)
	}
	return p
}

func (p KeyParams) MarshalText() (text []byte, err error) {
	// format: $argon2id$v=%d$m=%d,t=%d,p=%d,l=%d$<base64 salt>$
	if p.err != nil {
		return nil, p.err
	}
	var buf []byte
	// version
	buf = append(buf, "$argon2id$v="...)
	buf = strconv.AppendInt(buf, int64(argon2.Version), 10)
	// memory
	buf = append(buf, "$m="...)
	buf = strconv.AppendUint(buf, uint64(p.Memory), 10)
	// time
	buf = append(buf, ",t="...)
	buf = strconv.AppendUint(buf, uint64(p.Time), 10)
	// threads
	buf = append(buf, ",p="...)
	buf = strconv.AppendUint(buf, uint64(p.Threads), 10)
	// keyLen
	buf = append(buf, ",l="...)
	buf = strconv.AppendUint(buf, uint64(p.KeyLen), 10)
	// salt
	buf = append(buf, '$')
	buf = append(buf, base64Encode(p.Salt)...)
	buf = append(buf, '$')
	return buf, nil
}

func (p *KeyParams) UnmarshalText(text []byte) error {
	// format: $argon2id$v=%d$m=%d,t=%d,p=%d,l=%d$<base64 salt>$
	// parts[0] = <empty string>
	// parts[1] = argon2id
	// parts[2] = v=%d
	// parts[3] = m=%d,t=%d,p=%d,l=%d
	// parts[4] = <base64 salt>
	// parts[5] = <empty string>
	parts := strings.Split(string(text), "$")
	if len(parts) < 6 {
		return fmt.Errorf("invalid params")
	}
	var argon2Version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &argon2Version)
	if err != nil {
		return err
	}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d,l=%d", &p.Memory, &p.Time, &p.Threads, &p.KeyLen)
	if err != nil {
		return err
	}
	p.Salt, err = base64Decode([]byte(parts[4]))
	if err != nil {
		return err
	}
	return nil
}

func DeriveKey(password []byte, params KeyParams) (key []byte, err error) {
	if params.err != nil {
		return nil, params.err
	}
	return argon2.IDKey(password, params.Salt, params.Time, params.Memory, params.Threads, params.KeyLen), nil
}

func GenerateFromPassword(password []byte, params KeyParams) (passwordHash []byte, err error) {
	_, params.err = rand.Read(params.Salt)
	passwordHash, err = params.MarshalText()
	if err != nil {
		return nil, err
	}
	key, err := DeriveKey(password, params)
	if err != nil {
		return nil, err
	}
	passwordHash = append(passwordHash, base64Encode(key)...)
	return passwordHash, nil
}

func CompareHashAndPassword(passwordHash []byte, password []byte) error {
	i := bytes.LastIndex(passwordHash, []byte("$"))
	if i < 0 {
		return fmt.Errorf("invalid passwordHash")
	}
	var params KeyParams
	err := params.UnmarshalText(passwordHash[:i+1])
	if err != nil {
		return err
	}
	derivedKey, err := DeriveKey(password, params)
	if err != nil {
		return err
	}
	providedKey, err := base64Decode(passwordHash[i+1:])
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(providedKey, derivedKey) != 1 {
		return fmt.Errorf("incorrect password")
	}
	return nil
}
