package cryptoutil

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/bokwoon95/pagemanager/testutil"
)

func Test_Password(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		const password = "password"
		is := testutil.New(t, testutil.Parallel)
		hashedPassword, err := GenerateFromPassword([]byte(password), NewParams(nil))
		is.NoErr(err)
		fmt.Println(string(hashedPassword))
		err = CompareHashAndPassword(hashedPassword, []byte(password))
		is.NoErr(err)
	})
}

func Test_KeyDerivation(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		const password = "password"
		is := testutil.New(t, testutil.Parallel)
		params := NewParams(nil)
		fmt.Printf("%+v\n", params)
		key, err := DeriveKey([]byte(password), params)
		is.NoErr(err)
		var params2 Params
		b, err := params.MarshalText()
		is.NoErr(err)
		err = params2.UnmarshalText(b)
		is.NoErr(err)
		key2, err := DeriveKey([]byte(password), params2)
		is.NoErr(err)
		fmt.Printf("key : %s\n", hex.EncodeToString(key))
		fmt.Printf("key2: %s\n", hex.EncodeToString(key2))
		is.Equal(key, key2)
	})
}
