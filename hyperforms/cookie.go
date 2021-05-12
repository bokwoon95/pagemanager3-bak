package hyperforms

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"net/http"
	"time"

	"github.com/bokwoon95/pagemanager/cryptoutil"
)

const validationCookieName = "hyforms.ValidationErrMsgs"

var box *cryptoutil.KeyBox = func() *cryptoutil.KeyBox {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	box, err := cryptoutil.NewKeyBox(cryptoutil.StaticKey(key), nil)
	if err != nil {
		panic(err)
	}
	return box
}()

func SetCookieValue(w http.ResponseWriter, cookieName string, value interface{}, cookieTemplate *http.Cookie) error {
	buf := &bytes.Buffer{}
	switch value := value.(type) {
	case []byte:
		buf.Write(value)
	case string:
		buf.WriteString(value)
	default:
		err := gob.NewEncoder(buf).Encode(value)
		if err != nil {
			return err
		}
	}
	b64HashedValue, err := box.HashEncode(buf.Bytes())
	if err != nil {
		return err
	}
	cookie := &http.Cookie{}
	if cookieTemplate != nil {
		*cookie = *cookieTemplate
	}
	cookie.Path = "/"
	cookie.Name = cookieName
	cookie.Value = string(b64HashedValue)
	http.SetCookie(w, cookie)
	return nil
}

func GetCookieValue(w http.ResponseWriter, r *http.Request, cookieName string, dest interface{}) error {
	defer http.SetCookie(w, &http.Cookie{Path: "/", Name: cookieName, MaxAge: -1, Expires: time.Now().Add(-1 * time.Hour)})
	c, err := r.Cookie(cookieName)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return err
	}
	if c == nil {
		return nil
	}
	data, err := box.HashDecode([]byte(c.Value))
	if err != nil {
		return err
	}
	switch dest := dest.(type) {
	case *[]byte:
		*dest = data
	case *string:
		*dest = string(data)
	default:
		err = gob.NewDecoder(bytes.NewReader(data)).Decode(dest)
		if err != nil {
			return err
		}
	}
	return nil
}
