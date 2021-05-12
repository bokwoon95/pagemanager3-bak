package cryptoutil

import (
	"fmt"
	"testing"

	"github.com/bokwoon95/pagemanager/testutil"
)

func TestPasswordBox(t *testing.T) {
	const password = "abcdefg"
	is := testutil.New(t)
	box, err := NewPasswordBox(&pwstore{}, nil)
	is.NoErr(err)
	err = box.SetPassword([]byte(password))
	is.NoErr(err)
	is.True(box.PasswordEntered())
	err = box.EnterPassword([]byte(password))
	is.NoErr(err)
	is.True(box.PasswordEntered())

	t.Run("encryption", func(t *testing.T) {
		is := testutil.New(t)
		plaintext := []byte("lorem ipsum dolor sit amet")
		ciphertext, err := box.Encrypt(plaintext)
		is.NoErr(err)
		fmt.Println(string(ciphertext))
		got, err := box.Decrypt(ciphertext)
		is.NoErr(err)
		is.Equal(plaintext, got)
		_, err = box.Decrypt(append(ciphertext, "tampered"...))
		is.True(err != nil)
	})

	t.Run("hash", func(t *testing.T) {
		is := testutil.New(t)
		msg := []byte("lorem ipsum dolor sit amet")
		encodedMsg, err := box.HashEncode(msg)
		is.NoErr(err)
		fmt.Println(string(encodedMsg))
		got, err := box.HashDecode(encodedMsg)
		is.NoErr(err)
		is.Equal(msg, got)
		_, err = box.HashDecode(append(encodedMsg, "tampered"...))
		is.True(err != nil)
	})
}
