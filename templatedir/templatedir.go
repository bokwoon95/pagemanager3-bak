package templatedir

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dop251/goja"
)

var bufpool = sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}

var internalFS fs.FS

func init() {
	if internalFS == nil {
		_, currentfile, _, _ := runtime.Caller(0)
		currentdir := filepath.Dir(currentfile)
		internalFS = os.DirFS(currentdir)
	}
}

type TemplateDir struct {
	fsys             fs.FS
	store            ValueStore
	assetURLPrefix   string
	assetFilter      func(path string) (allow bool)
	assetNotFound    func(w http.ResponseWriter, r *http.Request)
	assetErrHandler  func(w http.ResponseWriter, r *http.Request, err error)
	fallbackAssets   map[string]string
	fallbackAssetsMu *sync.RWMutex
	configCache      map[string]templateConfig
	configCacheMu    *sync.RWMutex
	tmplCache        map[string]*template.Template
	tmplCacheMu      *sync.RWMutex
}

type Option func(*TemplateDir)

func AssetURLPrefix(urlprefix string) Option {
	return func(dir *TemplateDir) { dir.assetURLPrefix = urlprefix }
}

func AssetFilter(filter func(path string) (allow bool)) Option {
	return func(dir *TemplateDir) { dir.assetFilter = filter }
}

func AssetNotFound(notfound http.HandlerFunc) Option {
	return func(dir *TemplateDir) { dir.assetNotFound = notfound }
}

func AssetErrHandler(errhandler func(w http.ResponseWriter, r *http.Request, err error)) Option {
	return func(dir *TemplateDir) { dir.assetErrHandler = errhandler }
}

func New(fsys fs.FS, store ValueStore, opts ...Option) (*TemplateDir, error) {
	if fsys == nil {
		return nil, fmt.Errorf("dir cannot be nil")
	}
	if store == nil {
		return nil, fmt.Errorf("valueStore cannot be nil")
	}
	dir := &TemplateDir{
		fsys:             fsys,
		store:            store,
		fallbackAssets:   make(map[string]string),
		fallbackAssetsMu: &sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(dir)
	}
	dir.assetURLPrefix = strings.TrimSpace(dir.assetURLPrefix)
	if dir.assetURLPrefix == "" {
		dir.assetURLPrefix = "/templatedir/"
	}
	if !strings.HasPrefix(dir.assetURLPrefix, "/") {
		dir.assetURLPrefix = "/" + dir.assetURLPrefix
	}
	if !strings.HasSuffix(dir.assetURLPrefix, "/") {
		dir.assetURLPrefix = dir.assetURLPrefix + "/"
	}
	if dir.assetNotFound == nil {
		dir.assetNotFound = http.NotFound
	}
	if dir.assetErrHandler == nil {
		dir.assetErrHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	return dir, nil
}

func (dir *TemplateDir) Assets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, dir.assetURLPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		path := func() string {
			path := strings.TrimPrefix(r.URL.Path, dir.assetURLPrefix)
			ext := filepath.Ext(path)
			name := strings.TrimSuffix(path, ext)
			i := strings.LastIndex(name, ".")
			if i < 0 || !strings.HasPrefix(name[i+1:], "sha256-") {
				return path
			}
			return name[:i] + ext
		}()
		basepath := filepath.Base(path)
		if basepath == "config.js" || strings.HasSuffix(basepath, ".config.js") {
			dir.assetNotFound(w, r)
			return
		}
		if basepath == "editmode.js" || basepath == "editmode.css" || basepath == "env.js" {
			f, err := internalFS.Open(path)
			dir.serveFile(w, r, path, f, err)
			return
		}
		if dir.assetFilter != nil && !dir.assetFilter(path) {
			dir.assetNotFound(w, r)
			return
		}
		f, err := dir.fsys.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			f, err = func() (fs.File, error) {
				missingFile := "/" + path
				dir.fallbackAssetsMu.RLock()
				defer dir.fallbackAssetsMu.RUnlock()
				fallbackFile, ok := dir.fallbackAssets[missingFile]
				if !ok {
					return nil, os.ErrNotExist
				}
				return dir.fsys.Open(strings.TrimPrefix(fallbackFile, "/"))
			}()
		}
		dir.serveFile(w, r, path, f, err)
	})
}

func (dir *TemplateDir) serveFile(w http.ResponseWriter, r *http.Request, path string, f fs.File, err error) {
	if f != nil {
		defer f.Close()
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		dir.assetErrHandler(w, r, err)
		return
	}
	if f == nil {
		dir.assetNotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		dir.assetErrHandler(w, r, err)
		return
	}
	if info.IsDir() {
		dir.assetNotFound(w, r)
		return
	}
	fseeker, ok := f.(io.ReadSeeker)
	if !ok {
		b, err := io.ReadAll(f)
		if err != nil {
			dir.assetErrHandler(w, r, err)
			return
		}
		fseeker = bytes.NewReader(b)
	}
	http.ServeContent(w, r, path, info.ModTime(), fseeker)
}

type ValueStore interface {
	GetValue(localeCode, namespace, name string) (value NullString, err error)
	GetRows(localeCode, namespace, name string) (rows []map[string]interface{}, err error)
	BeginTx() (ValueStoreTx, error)
}

type ValueStoreTx interface {
	SetValue(localeCode, namespace, name string, value string) error
	SetRows(localeCode, namespace, name string, rows []map[string]interface{}) error
	Commit() error
	Rollback() error
}

type templateData struct {
	URL            string
	Namespace      string
	LocaleCode     string
	EditMode       bool
	Vars           map[string]interface{}
	css            []string
	js             []string
	csp            map[string][]string
	fsys           fs.FS
	assetURLPrefix string
	// CSS/JS are methods that can either return just the path or inline the script entirely (because templatedata retains a reference to the fs.FS). This means there is no need for a Data []byte.
}

type serveConfig struct {
	disableCSP     bool
	localeCode     string
	editMode       bool
	css            []string
	js             []string
	bufferResponse bool
	cacheConfig    bool
}

type ServeOption func(*serveConfig)

func (dir *TemplateDir) ServeTemplate(w io.Writer, r *http.Request, subDir, templateConfigPath string, opts ...ServeOption) error {
	var data templateData
	var config serveConfig
	for _, opt := range opts {
		opt(&config)
	}
	data.URL = r.URL.Path
	data.Namespace = r.URL.Path
	data.LocaleCode = config.localeCode
	data.EditMode = config.editMode
	data.css = append(data.css, config.css...)
	data.js = append(data.js, config.js...)
	data.fsys = dir.fsys
	data.assetURLPrefix = dir.assetURLPrefix
	subDir = strings.TrimPrefix(strings.TrimSuffix(subDir, "/"), "/")
	var configjs interface{}
	b, err := fs.ReadFile(dir.fsys, subDir+"/config.js")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			configjs = map[string]interface{}{}
		} else {
			return err
		}
	} else {
		vm := goja.New()
		val, err := vm.RunString(`(function(){` + string(b) + `})()`)
		if err != nil {
			return err
		}
		configjs = val.Export()
	}
	b, err = fs.ReadFile(dir.fsys, subDir+"/"+templateConfigPath)
	if err != nil {
		return err
	}
	vm := goja.New()
	vm.Set("$CONFIG", configjs)
	val, err := vm.RunString(`(function(){` + string(b) + `})()`)
	if err != nil {
		return err
	}
	var tconfig templateConfig
	err = tconfig.Unmarshal(subDir, val.Export())
	if err != nil {
		return err
	}
	data.css = append(data.css, tconfig.css...)
	data.js = append(data.js, tconfig.js...)
	data.Vars = tconfig.vars
	data.csp = tconfig.contentSecurityPolicy
	spew.Dump(data)
	if len(tconfig.html) == 0 {
		return fmt.Errorf("no files provided")
	}
	t := template.New("").Funcs(dir.funcs())
	for _, html := range tconfig.html {
		html = strings.TrimPrefix(html, "/")
		b, err = fs.ReadFile(dir.fsys, html)
		if err != nil {
			return err
		}
		_, err = t.New(html).Parse(string(b))
		if err != nil {
			return err
		}
	}
	err = t.ExecuteTemplate(w, strings.TrimPrefix(tconfig.html[0], "/"), data)
	return err
}

func (data templateData) CSS() (template.HTML, error) {
	buf := bufpool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufpool.Put(buf)
	}()
	for _, css := range data.css {
		if strings.HasPrefix(css, "/") {
			buf.WriteString("\n" + `<script src="` + css + `"></script>`)
		} else {
			buf.WriteString("\n" + `<script src="` + data.assetURLPrefix + css + `"></script>`)
		}
	}
	return template.HTML(buf.String()), nil
}

func (data templateData) JS() (template.HTML, error) {
	buf := bufpool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufpool.Put(buf)
	}()
	buf.WriteString(`<script type="application/json" data-env>`)
	err := json.NewEncoder(buf).Encode(map[string]interface{}{
		"URL":        data.URL,
		"Namespace":  data.Namespace,
		"LocaleCode": data.LocaleCode,
		"EditMode":   data.EditMode,
	})
	if err != nil {
		return "", err
	}
	buf.WriteString(`</script>`)
	buf.WriteString("\n" + `<script src="` + data.assetURLPrefix + `env.js"></script>`)
	for _, js := range data.js {
		if strings.HasPrefix(js, "/") {
			buf.WriteString("\n" + `<script src="` + js + `"></script>`)
		} else {
			buf.WriteString("\n" + `<script src="` + data.assetURLPrefix + js + `"></script>`)
		}
	}
	return template.HTML(buf.String()), nil
}

func (data templateData) ContentSecurityPolicy() (template.HTML, error) {
	return "", nil
}

func (dir *TemplateDir) funcs() map[string]interface{} {
	return map[string]interface{}{
		"getValue": func(data templateData, name string, opts ...func(data *templateData)) (value NullString, err error) {
			for _, opt := range opts {
				opt(&data)
			}
			return dir.store.GetValue(data.LocaleCode, data.Namespace, name)
		},
		"getRows": func(data templateData, name string, opts ...func(data *templateData)) (rows []map[string]interface{}, err error) {
			for _, opt := range opts {
				opt(&data)
			}
			return dir.store.GetRows(data.LocaleCode, data.Namespace, name)
		},
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"namespace": func(namespace string) func(data *templateData) {
			return func(data *templateData) { data.Namespace = namespace }
		},
		"localeCode": func(localeCode string) func(data *templateData) {
			return func(data *templateData) { data.LocaleCode = localeCode }
		},
	}
}

type templateConfig struct {
	html                  []string
	css                   []string
	js                    []string
	vars                  map[string]interface{}
	contentSecurityPolicy map[string][]string
}

func (cfg *templateConfig) Unmarshal(subDir string, v interface{}) error {
	m, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf("not a map")
	}
	if __list__, ok := m["HTML"].([]interface{}); ok {
		for _, __html__ := range __list__ {
			if html, ok := __html__.(string); ok {
				if strings.HasPrefix(html, "/") {
					cfg.html = append(cfg.html, html)
				} else {
					cfg.html = append(cfg.html, subDir+"/"+html)
				}
			}
		}
	}
	if __list__, ok := m["CSS"].([]interface{}); ok {
		for _, __css__ := range __list__ {
			if css, ok := __css__.(string); ok {
				if strings.HasPrefix(css, "/") {
					cfg.css = append(cfg.css, css)
				} else {
					cfg.css = append(cfg.css, subDir+"/"+css)
				}
			}
		}
	}
	if __list__, ok := m["JS"].([]interface{}); ok {
		for _, __js__ := range __list__ {
			if js, ok := __js__.(string); ok {
				if strings.HasPrefix(js, "/") {
					cfg.js = append(cfg.js, js)
				} else {
					cfg.js = append(cfg.js, subDir+"/"+js)
				}
			}
		}
	}
	cfg.vars, _ = m["Vars"].(map[string]interface{})
	if __csp__, ok := m["ContentSecurityPolicy"].(map[string][]interface{}); ok {
		if cfg.contentSecurityPolicy == nil {
			cfg.contentSecurityPolicy = make(map[string][]string)
		}
		for policy, __values__ := range __csp__ {
			var values []string
			for _, __value__ := range __values__ {
				if value, ok := __value__.(string); ok {
					values = append(values, value)
				}
			}
			cfg.contentSecurityPolicy[policy] = values
		}
	}
	return nil
}

type NullString struct {
	Valid bool
	Str   string
}

// Scan implements the Scanner interface.
func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.Str, ns.Valid = "", false
		return nil
	}
	switch value := value.(type) {
	case int64:
		ns.Str = strconv.FormatInt(value, 10)
	case float64:
		ns.Str = strconv.FormatFloat(value, 'g', -1, 64)
	case bool:
		ns.Str = strconv.FormatBool(value)
	case []byte:
		ns.Str = string(value)
	case string:
		ns.Str = value
	case time.Time:
		ns.Str = value.Format(time.RFC3339Nano)
	}
	ns.Valid = true
	return nil
}

// Value implements the driver Valuer interface.
func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.Str, nil
}

func (ns NullString) String() string {
	return ns.Str
}
