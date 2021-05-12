package hyperforms

import (
	"bytes"
	"strconv"

	hy "github.com/bokwoon95/pagemanager/hypergo"
)

type FormInput interface {
	hy.Element
	ID() string
	Name() string
	Value() string
	Values() []string
	ErrMsgs() []string
}

type input struct {
	form  *Form
	attrs hy.Attributes
	name  string
}

func (i input) ID() string { return i.attrs.ID }

func (i input) Name() string { return i.name }

func (i input) Value() string { return i.form.Request.FormValue(i.name) }

func (i input) Values() []string { return i.form.Request.Form[i.name] }

func (i input) ErrMsgs() []string { return i.form.InputErrMsgs[i.name] }

type Input struct {
	input
	inputType    string
	defaultValue string
}

func (i Input) WriteHTML(buf *bytes.Buffer, sanitizer hy.Sanitizer) error {
	if i.attrs.Dict == nil {
		i.attrs.Dict = make(map[string]string)
	}
	i.attrs.Dict["name"] = i.name
	var children []hy.Element
	switch i.inputType {
	case "textarea":
		i.attrs.Tag = "textarea"
		children = append(children, hy.Txt(i.defaultValue))
		delete(i.attrs.Dict, "type")
		delete(i.attrs.Dict, "value")
	default:
		i.attrs.Tag = "input"
		i.attrs.Dict["type"] = i.inputType
		i.attrs.Dict["value"] = i.defaultValue
	}
	return hy.WriteHTML(buf, i.attrs, children, sanitizer)
}

func (f *Form) Input(inputType string, name string, defaultValue string) Input {
	return Input{input: input{form: f, name: name}, inputType: inputType, defaultValue: defaultValue}
}

func (f *Form) Text(name string, defaultValue string) Input {
	return Input{input: input{form: f, name: name}, inputType: "text", defaultValue: defaultValue}
}

func (f *Form) Hidden(name string, defaultValue string) Input {
	return Input{input: input{form: f, name: name}, inputType: "hidden", defaultValue: defaultValue}
}

func (f *Form) Textarea(name string, defaultValue string) Input {
	return Input{input: input{form: f, name: name}, inputType: "textarea", defaultValue: defaultValue}
}

func (i *Input) Set(selector string, attributes map[string]string) {
	i.attrs = hy.ParseAttributes(selector, attributes)
}

func (i Input) Int(validators ...Validator) (num int, err error) {
	value := i.form.Request.FormValue(i.name)
	num, err = strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	validateInput(i.form, i.name, num, validators)
	return num, nil
}

func (i Input) Float64(validators ...Validator) (num float64, err error) {
	value := i.form.Request.FormValue(i.name)
	num, err = strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	validateInput(i.form, i.name, num, validators)
	return num, nil
}

func (i Input) Validate(validators ...Validator) Input {
	validateInput(i.form, i.name, i.form.Request.FormValue(i.name), validators)
	return i
}

type ToggledInput struct {
	input
	inputType string
	value     string
	checked   bool
}

func (i ToggledInput) WriteHTML(buf *bytes.Buffer, sanitizer hy.Sanitizer) error {
	if i.attrs.Dict == nil {
		i.attrs.Dict = make(map[string]string)
	}
	i.attrs.Tag = "input"
	i.attrs.Dict["type"] = i.inputType
	i.attrs.Dict["name"] = i.name
	delete(i.attrs.Dict, "value")
	if i.value != "" {
		i.attrs.Dict["value"] = i.value
	}
	delete(i.attrs.Dict, "checked")
	if i.checked {
		i.attrs.Dict["checked"] = hy.Enabled
	} else {
		i.attrs.Dict["checked"] = hy.Disabled
	}
	return hy.WriteHTML(buf, i.attrs, nil, sanitizer)
}

func (i *ToggledInput) Set(selector string, attributes map[string]string) {
	i.attrs = hy.ParseAttributes(selector, attributes)
}

func (i *ToggledInput) Check(b bool) {
	i.checked = b
}

func (i ToggledInput) Checked() bool {
	for _, v := range i.form.Request.Form[i.name] {
		if i.value == "" && v == "on" {
			return true
		}
		if i.value != "" && v == i.value {
			return true
		}
	}
	return false
}

type ToggledInputs struct {
	input
	inputType string
	Options   []string
}

func (f *Form) Checkboxes(name string, options []string) ToggledInputs {
	return ToggledInputs{input: input{form: f, name: name}, inputType: "checkbox", Options: options}
}

func (f *Form) Radios(name string, options []string) ToggledInputs {
	return ToggledInputs{input: input{form: f, name: name}, inputType: "radio", Options: options}
}

func (i ToggledInputs) Inputs() []ToggledInput {
	var inputs []ToggledInput
	for _, option := range i.Options {
		inputs = append(inputs, ToggledInput{input: input{form: i.form, name: i.name}, inputType: i.inputType, value: option})
	}
	return inputs
}

type Option struct {
	Value    string
	Display  string
	Disabled bool
	Selected bool
	Optgroup string
	Options  []Option
}

func (opt Option) WriteHTML(buf *bytes.Buffer, sanitizer hy.Sanitizer) error {
	attrs := hy.Attributes{
		Tag:  "option",
		Dict: map[string]string{"value": opt.Value},
	}
	if opt.Disabled {
		attrs.Dict["disabled"] = hy.Enabled
	}
	if opt.Selected {
		attrs.Dict["selected"] = hy.Enabled
	}
	err := hy.WriteHTML(buf, attrs, []hy.Element{hy.Txt(opt.Display)}, sanitizer)
	if err != nil {
		return err
	}
	return nil
}

type SelectInput struct {
	input
	Options []Option
}

func (i SelectInput) WriteHTML(buf *bytes.Buffer, sanitizer hy.Sanitizer) error {
	if i.attrs.Dict == nil {
		i.attrs.Dict = make(map[string]string)
	}
	if i.attrs.ParseErr != nil {
		return i.attrs.ParseErr
	}
	buf.WriteString(`<select`)
	i.attrs.Tag = "select"
	i.attrs.Dict["name"] = i.name
	hy.WriteAttributes(buf, i.attrs, sanitizer)
	buf.WriteString(`>`)
	var err error
	for _, opt := range i.Options {
		switch opt.Optgroup {
		case "":
			err = opt.WriteHTML(buf, sanitizer)
			if err != nil {
				return err
			}
		default:
			attrs := hy.Attributes{
				Tag:  "optgroup",
				Dict: map[string]string{"label": opt.Optgroup},
			}
			if opt.Disabled {
				attrs.Dict["disabled"] = hy.Enabled
			}
			if opt.Selected {
				attrs.Dict["selected"] = hy.Enabled
			}
			var children []hy.Element
			for _, option := range opt.Options {
				if len(option.Options) > 0 {
					continue
				}
				children = append(children, option)
			}
			err = hy.WriteHTML(buf, attrs, children, sanitizer)
			if err != nil {
				return err
			}
		}
	}
	buf.WriteString(`</select>`)
	return nil
}

func (f *Form) Select(name string, options []Option) SelectInput {
	return SelectInput{input: input{form: f, name: name}, Options: options}
}

func (i *SelectInput) Set(selector string, attributes map[string]string) {
	i.attrs = hy.ParseAttributes(selector, attributes)
}

func (i *SelectInput) AppendOptions(opts ...Option) {
	i.Options = append(i.Options, opts...)
}
