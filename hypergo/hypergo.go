package hy

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"sync"
)

const (
	Enabled  = "\x00"
	Disabled = "\x01"
)

type Sanitizer = func(tag, attrName, attrValue string) (allow bool)

type Element interface {
	WriteHTML(buf *bytes.Buffer, sanitizer Sanitizer) error
}

var bufpool = sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}

type Attr map[string]string

type Attributes struct {
	ParseErr error
	Tag      string
	ID       string
	Classes  []string
	Dict     map[string]string
}

func ParseAttributes(selector string, attributes map[string]string) Attributes {
	type State uint8
	const (
		StateNone State = iota
		StateTag
		StateID
		StateClass
		StateAttrName
		StateAttrValue
	)
	attrs := Attributes{Dict: attributes}
	state := StateTag
	var name []rune
	var value []rune
	for i, c := range selector {
		if c == '#' || c == '.' || c == '[' {
			switch state {
			case StateTag:
				attrs.Tag = string(value)
			case StateID:
				attrs.ID = string(value)
			case StateClass:
				if len(value) > 0 {
					attrs.Classes = append(attrs.Classes, string(value))
				}
			case StateAttrName, StateAttrValue:
				attrs.ParseErr = fmt.Errorf("unclosed attribute: position=%d char=%c selector=%s", i, c, selector)
				return attrs
			}
			value = value[:0]
			switch c {
			case '#':
				state = StateID
			case '.':
				state = StateClass
			case '[':
				state = StateAttrName
			}
			continue
		}
		if c == '=' {
			switch state {
			case StateAttrName:
				state = StateAttrValue
			default:
				attrs.ParseErr = fmt.Errorf("unopened attribute: position=%d char=%c selector=%s", i, c, selector)
				return attrs
			}
			continue
		}
		if c == ']' {
			if state == StateAttrName || state == StateAttrValue {
				if _, ok := attrs.Dict[string(name)]; !ok {
					if attrs.Dict == nil {
						attrs.Dict = make(map[string]string)
					}
					switch state {
					case StateAttrName:
						attrs.Dict[string(name)] = Enabled
					case StateAttrValue:
						attrs.Dict[string(name)] = string(value)
					}
				}
			} else {
				attrs.ParseErr = fmt.Errorf("unopened attribute: position=%d char=%c selector=%s", i, c, selector)
				return attrs
			}
			name = name[:0]
			value = value[:0]
			state = StateNone
			continue
		}
		switch state {
		case StateTag, StateID, StateClass, StateAttrValue:
			value = append(value, c)
		case StateAttrName:
			name = append(name, c)
		case StateNone:
			attrs.ParseErr = fmt.Errorf("unknown state (please prepend with '#', '.' or '['): position=%d char=%c selector=%s", i, c, selector)
			return attrs
		}
	}
	// flush value
	if len(value) > 0 {
		switch state {
		case StateTag:
			attrs.Tag = string(value)
		case StateID:
			attrs.ID = string(value)
		case StateClass:
			attrs.Classes = append(attrs.Classes, string(value))
		case StateNone: // do nothing i.e. drop the value
		case StateAttrName, StateAttrValue:
			attrs.ParseErr = fmt.Errorf("unclosed attribute: selector=%s, value: %s", selector, string(value))
			return attrs
		}
		value = value[:0]
	}
	if id, ok := attrs.Dict["id"]; ok {
		delete(attrs.Dict, "id")
		attrs.ID = id
	}
	if class, ok := attrs.Dict["class"]; ok {
		delete(attrs.Dict, "class")
		for _, c := range strings.Split(class, " ") {
			if c == "" {
				continue
			}
			attrs.Classes = append(attrs.Classes, c)
		}
	}
	return attrs
}

func (attrs *Attributes) AddClasses(classes ...string) {
	attrs.Classes = append(attrs.Classes, classes...)
}

func (attrs *Attributes) RemoveClasses(classes ...string) {
	excluded := make(map[string]struct{})
	for _, class := range classes {
		excluded[class] = struct{}{}
	}
	classes = attrs.Classes[:0]
	for _, class := range attrs.Classes {
		if _, ok := excluded[class]; ok {
			continue
		}
		classes = append(classes, class)
	}
	attrs.Classes = attrs.Classes[:len(classes)]
}

func (attrs *Attributes) SetAttribute(name, value string) {
	if strings.EqualFold(name, "id") {
		attrs.ID = value
		return
	}
	if strings.EqualFold(name, "class") {
		classes := strings.Split(value, " ")
		attrs.Classes = attrs.Classes[:0]
		for _, class := range classes {
			if class == "" {
				continue
			}
			attrs.Classes = append(attrs.Classes, class)
		}
		return
	}
	if attrs.Dict == nil {
		attrs.Dict = make(map[string]string)
	}
	attrs.Dict[name] = value
}

func (attrs *Attributes) RemoveAttribute(name string) {
	if strings.EqualFold(name, "id") {
		attrs.ID = ""
		return
	}
	if strings.EqualFold(name, "class") {
		attrs.Classes = attrs.Classes[:0]
		return
	}
	delete(attrs.Dict, name)
}

// https://developer.mozilla.org/en-US/docs/Glossary/Empty_element
var singletonElements = map[string]struct{}{
	"area": {}, "base": {}, "br": {}, "col": {}, "embed": {}, "hr": {}, "img": {}, "input": {},
	"link": {}, "meta": {}, "param": {}, "source": {}, "track": {}, "wbr": {},
}

func WriteHTML(buf *bytes.Buffer, attrs Attributes, children []Element, sanitizer Sanitizer) error {
	if sanitizer == nil {
		sanitizer = DefaultSanitizer
	}
	if attrs.ParseErr != nil {
		return attrs.ParseErr
	}
	if attrs.Tag == "" {
		attrs.Tag = "div"
	}
	if !sanitizer(attrs.Tag, "", "") {
		return nil
	}
	buf.WriteString(`<` + attrs.Tag)
	WriteAttributes(buf, attrs, sanitizer)
	buf.WriteString(`>`)
	if _, ok := singletonElements[strings.ToLower(attrs.Tag)]; ok {
		return nil
	}
	var err error
	for _, child := range children {
		if child == nil {
			continue
		}
		err = child.WriteHTML(buf, sanitizer)
		if err != nil {
			return err
		}
	}
	buf.WriteString(`</` + attrs.Tag + `>`)
	return nil
}

func WriteAttributes(buf *bytes.Buffer, attrs Attributes, sanitizer Sanitizer) {
	if sanitizer == nil {
		sanitizer = DefaultSanitizer
	}
	if attrs.ID != "" && sanitizer(attrs.Tag, "id", attrs.ID) {
		buf.WriteString(` id="`)
		escapeHTML(buf, htmlReplacementTable, attrs.ID)
		buf.WriteString(`"`)
	}
	if class := strings.Join(attrs.Classes, " "); class != "" && sanitizer(attrs.Tag, "class", class) {
		buf.WriteString(` class="`)
		escapeHTML(buf, htmlReplacementTable, class)
		buf.WriteString(`"`)
	}
	var names []string
	for name := range attrs.Dict {
		if strings.EqualFold(name, "id") || strings.EqualFold(name, "class") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var value string
	for _, name := range names {
		value = attrs.Dict[name]
		if name == "" || value == Disabled {
			continue
		}
		if !sanitizer(attrs.Tag, name, value) {
			continue
		}
		buf.WriteString(` `)
		escapeHTML(buf, attrNameReplacementTable, name)
		if value == Enabled {
			continue
		}
		buf.WriteString(`="`)
		if isURLAttr(name) {
			escapeURL(buf, false, value)
		} else if strings.EqualFold(name, "srcset") {
			sanitizeAndEscapeSrcset(buf, value)
		} else {
			escapeHTML(buf, htmlReplacementTable, value)
		}
		buf.WriteString(`"`)
	}
}

func Marshal(sanitizer Sanitizer, el Element) (template.HTML, error) {
	if el == nil {
		return "", nil
	}
	if sanitizer == nil {
		sanitizer = DefaultSanitizer
	}
	buf := bufpool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufpool.Put(buf)
	}()
	err := el.WriteHTML(buf, sanitizer)
	output := template.HTML(buf.String())
	if err != nil {
		return output, fmt.Errorf("hypergo: Marshal failed: %w", err)
	}
	return output, nil
}
