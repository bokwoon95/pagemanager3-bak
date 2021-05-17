package templatedir

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bokwoon95/pagemanager/testutil"
	"github.com/dop251/goja"
)

const jsCfg = `return {
  Vars: {
    Namespace: "bokwoon95/plainsimple",
  },
  Fun: function() { return 5 },
  ContentSecurityPolicy: {
    "script-src": ["stackpath.bootstrapcdn.com", "code.jquery.com"],
    "style-src": ["stackpath.bootstrapcdn.com", "fonts.googleapis.com"],
    "img-src": ["source.unsplash.com", "images.unsplash.com"],
    "font-src": ["fonts.gstatic.com"],
  },
};`

const jsThemeCfg = `return {
  Name: "plainsimple",
  Description: "Just a plain simple theme",
  FallbackAssets: {
    "/pm-images/plainsimple/hero.jpg": "hero.jpg",
    "/pm-images/plainsimple/face.jpg": "face.jpg",
  },
  Vars: $CONFIG.Vars,
  Woh: $CONFIG.Fun(),
};`

// templatedir.New(dir, store, AssetsDir("/bruh", nil))
// router.Use(templatedir.Assets)

func Test_Z(t *testing.T) {
	is := testutil.New(t)
	vm := goja.New()
	val, err := vm.RunString(`(function(){` + jsCfg + `})()`)
	is.NoErr(err)
	fmt.Println(val.Export())
	vm2 := goja.New()
	vm2.Set("$CONFIG", val.Export())
	val2, err := vm2.RunString(`(function(){` + jsThemeCfg + `})()`)
	is.NoErr(err)
	fmt.Println(val2.Export())
}

func Test_ServeTemplate(t *testing.T) {
	is := testutil.New(t)
	_, currentfile, _, _ := runtime.Caller(0)
	themesdir := filepath.Join(filepath.Dir(filepath.Dir(currentfile)), "pm-themes")
	store := newVstore()
	dir, err := New(os.DirFS(themesdir), store)
	is.NoErr(err)
	r, err := http.NewRequest("GET", "/hello", nil)
	is.NoErr(err)
	err = dir.ServeTemplate(os.Stdout, r, "plainsimple", "index.config.js")
	is.NoErr(err)
}

func Test_Assets(t *testing.T) {
	is := testutil.New(t)
	_, currentfile, _, _ := runtime.Caller(0)
	themesdir := filepath.Join(filepath.Dir(filepath.Dir(currentfile)), "pm-themes")
	store := newVstore()
	dir, err := New(os.DirFS(themesdir), store)
	is.NoErr(err)
	newrequest := func(url string) *http.Request {
		r, _ := http.NewRequest("GET", url, nil)
		return r
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path, "next called")
	})
	is.NoErr(err)
	var rr *httptest.ResponseRecorder
	for _, url := range []string{
		"/templatedir/env.js",
		"/templatedir/plainsimple/config.js",
		"/templatedir/plainsimple/index.config.js",
		"/templatedir/plainsimple/index.html",
		"/templatedir/plainsimple/index.css",
	} {
		rr = httptest.NewRecorder()
		dir.Assets(next).ServeHTTP(rr, newrequest(url))
		fmt.Println(url, rr.Result().Status, rr.Body.String())
	}
}

// runtime options:
// disable CSP?
// set localeCode?
// set editMode?
// add CSS paths?
// add JS paths?
// buffer response?
// cache config?
//
// data-namespace
// data-name
// data-row
// data-row.name
// data-row.href
