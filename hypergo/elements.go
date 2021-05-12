package hy

import "bytes"

type Elements []Element

func (l Elements) WriteHTML(buf *bytes.Buffer, sanitizer Sanitizer) error {
	if sanitizer == nil {
		sanitizer = DefaultSanitizer
	}
	var err error
	for _, el := range l {
		if el == nil {
			continue
		}
		err = el.WriteHTML(buf, sanitizer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Elements) Append(selector string, attributes map[string]string, children ...Element) {
	*l = append(*l, H(selector, attributes, children...))
}

func (l *Elements) AppendElements(children ...Element) {
	*l = append(*l, children...)
}
