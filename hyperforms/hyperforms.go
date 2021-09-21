package hyperforms

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"time"

	hy "github.com/bokwoon95/pagemanager/hypergo"
)

// TODO: I've figured out how to use errors instead of errmsgs. This package
// needs its own custom NewError() constructor that returns an error that can
// be gob encoded. The custom error is based on strings, not pointers, so
// errors of the same string are fungible. If the user passes in an
// errors.errorString instead

type Form struct {
	attrs        hy.Attributes
	children     []hy.Element
	Request      *http.Request
	ErrMsgs      []string
	InputErrMsgs map[string][]string
}

func New(w http.ResponseWriter, r *http.Request) *Form {
	form := &Form{Request: r}
	func() {
		if r == nil || r.Method != "GET" {
			return
		}
		c, _ := r.Cookie(validationCookieName)
		if c == nil {
			return
		}
		defer http.SetCookie(w, &http.Cookie{Name: validationCookieName, MaxAge: -1})
		b, err := box.HashDecode([]byte(c.Value))
		if err != nil {
			return
		}
		validationErr := validationErrMsgs{}
		err = gob.NewDecoder(bytes.NewReader(b)).Decode(&validationErr)
		if err != nil {
			return
		}
		if time.Now().After(validationErr.Expires) {
			return
		}
		form.ErrMsgs = validationErr.FormErrMsgs
		form.InputErrMsgs = validationErr.InputErrMsgs
	}()
	return form
}

func (f *Form) WriteHTML(buf *bytes.Buffer, sanitizer hy.Sanitizer) error {
	// TODO: check f.request.Context() for any CSRF token and prepend it into the form as necessary
	// or should this be done in a hook?
	f.attrs.Tag = "form"
	err := hy.WriteHTML(buf, f.attrs, f.children, sanitizer)
	if err != nil {
		return err
	}
	return nil
}

func (f *Form) AddClasses(classes ...string) { f.attrs.AddClasses(classes...) }

func (f *Form) RemoveClasses(classes ...string) { f.attrs.RemoveClasses(classes...) }

func (f *Form) SetAttribute(name, value string) { f.attrs.SetAttribute(name, value) }

func (f *Form) RemoveAttribute(name string) { f.attrs.RemoveAttribute(name) }

func (f *Form) Append(selector string, attributes map[string]string, children ...hy.Element) {
	f.children = append(f.children, hy.H(selector, attributes, children...))
}

func (f *Form) AppendElements(children ...hy.Element) {
	f.children = append(f.children, children...)
}

func (f *Form) Marshal() (template.HTML, error) { return hy.Marshal(nil, f) }

func (f *Form) AddErrMsgs(errMsgs ...string) {
	f.ErrMsgs = append(f.ErrMsgs, errMsgs...)
}

func (f *Form) AddInputErrMsgs(inputName string, errMsgs ...string) {
	f.InputErrMsgs[inputName] = append(f.InputErrMsgs[inputName], errMsgs...)
}

func (f *Form) Redirect(w http.ResponseWriter, r *http.Request, url string) error {
	defer http.Redirect(w, r, url, http.StatusMovedPermanently)
	if r.Method == "GET" {
		return nil
	}
	errMsgs := validationErrMsgs{
		FormErrMsgs:  f.ErrMsgs,
		InputErrMsgs: f.InputErrMsgs,
		Expires:      time.Now().Add(10 * time.Second),
	}
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(errMsgs)
	if err != nil {
		return fmt.Errorf("%+v: failed gob encoding %s", errMsgs, err.Error())
	}
	value, err := box.HashEncode(buf.Bytes())
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:   validationCookieName,
		Value:  string(value),
		MaxAge: 10,
	})
	return nil
}
