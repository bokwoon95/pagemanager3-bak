package hy

import (
	"html/template"
	"testing"

	"github.com/bokwoon95/pagemanager/testutil"
)

func Test_H(t *testing.T) {
	assert := func(t *testing.T, el Element, want string) {
		is := testutil.New(t, testutil.Parallel, testutil.FailFast)
		got, err := Marshal(nil, el)
		is.NoErr(err)
		is.Equal(template.HTML(want), got)
	}
	t.Run("empty", func(t *testing.T) {
		assert(t, H("", nil), `<div></div>`)
	})
	t.Run("tag only", func(t *testing.T) {
		assert(t, H("div", nil), `<div></div>`)
	})
	t.Run("selector tags, id, classes and attributes", func(t *testing.T) {
		el := H(
			`p#id1.class1.class2.class3#id2`+
				`[accesskey=a b c]`+
				`[autocapitalize=sentences]`+
				`[contenteditable=true]`+
				`[hidden]`,
			nil,
			Txt(`<script>alert(1)</script>`),
		)
		want := `<p` +
			` id="id2"` +
			` class="class1 class2 class3"` +
			` accesskey="a b c"` +
			` autocapitalize="sentences"` +
			` contenteditable="true"` +
			` hidden` +
			`>&lt;script&gt;alert(1)&lt;/script&gt;</p>`
		assert(t, el, want)
	})
	t.Run("attributes overwrite selector", func(t *testing.T) {
		el := H(
			`p#id1.class1.class2.class3#id2`+
				`[accesskey=a b c]`+
				`[autocapitalize=sentences]`+
				`[contenteditable=true]`+
				`[hidden]`,
			Attr{
				"id":              "id3",
				"class":           "class4 class5 class6",
				"accesskey":       "d e f",
				"autocapitalize":  "words",
				"contenteditable": "false",
				"hidden":          Disabled,
			},
			UnsafeTxt(`<script>alert(1)</script>`),
		)
		want := `<p` +
			` id="id3"` +
			` class="class1 class2 class3 class4 class5 class6"` +
			` accesskey="d e f"` +
			` autocapitalize="words"` +
			` contenteditable="false"` +
			`><script>alert(1)</script></p>`
		assert(t, el, want)
	})
}
