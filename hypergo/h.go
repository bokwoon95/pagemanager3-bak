package hy

import (
	"bytes"
)

type HTMLElement struct {
	attrs    Attributes
	children []Element
}

func (el HTMLElement) WriteHTML(buf *bytes.Buffer, sanitizer Sanitizer) error {
	return WriteHTML(buf, el.attrs, el.children, sanitizer)
}

func H(selector string, attributes map[string]string, children ...Element) HTMLElement {
	return HTMLElement{
		attrs:    ParseAttributes(selector, attributes),
		children: children,
	}
}

func (el *HTMLElement) Set(selector string, attributes map[string]string, children ...Element) {
	el.attrs = ParseAttributes(selector, attributes)
	el.children = children
}

func (el *HTMLElement) Append(selector string, attributes map[string]string, children ...Element) {
	el.children = append(el.children, H(selector, attributes, children...))
}

func (el *HTMLElement) AppendElements(elements ...Element) {
	el.children = append(el.children, elements...)
}

func (el *HTMLElement) AddClasses(classes ...string) { el.attrs.AddClasses(classes...) }

func (el *HTMLElement) RemoveClasses(classes ...string) { el.attrs.RemoveClasses(classes...) }

func (el *HTMLElement) SetAttribute(name, value string) { el.attrs.SetAttribute(name, value) }

func (el *HTMLElement) RemoveAttribute(name string) { el.attrs.RemoveAttribute(name) }

func (el HTMLElement) Tag() string { return el.attrs.Tag }

func (el HTMLElement) ID() string { return el.attrs.ID }
