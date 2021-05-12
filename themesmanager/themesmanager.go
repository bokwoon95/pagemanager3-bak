package themesmanager

import "io/fs"

type ThemesManager struct {
	fsys fs.FS
}

type Asset struct {
	Path   string
	Data   []byte
	Hash   [32]byte
	Inline bool
}

type Template struct {
	HTML                  []string
	CSS                   []Asset
	JS                    []Asset
	TemplateVariables     map[string]interface{}
	ContentSecurityPolicy map[string][]string
}

type Theme struct {
	path           string // path to the theme folder in the "pm-themes" folder
	name           string
	description    string
	fallbackAssets map[string]string
	templates      map[string]Template
}

type TemplateData struct {
}

// .Ctx // <-- is this still needed?
// .URL
// .CSS
// .JS
// .CSP
// .DataID
// .LocaleCode
// .EditMode
// getValue
// getRows
// safeHTML
// localeCode
// dataID
// {{ getValue . "title" }}
// {{ getValue . "title" (localeCode "") (dataID "") }}

// if themesmanager is a package external to pagemanager, how do I pass the locale down to it? Declare themesmanager-specific localeCode context key so that pagemanager can populate it? But that would mean localeCode context is duplicated for both pagemanager and themesmanager.
// How about when I call ServeTemplate I must explicitly pass in the LocaleCode? I think that's better.
// LocaleCode(code string) and EditMode(bool) are both TemplateOptions that I can optionally pass in when calling ServeTemplate

// This means themesmanager is responsible for the editmode.js asset, which means it needs to somehow serve that at a specific endpoint and communicate it down the templates which must all reflect that endpoint.
// Is this too much coupling?
// Should I just merge themesmanager into pagemanager?
// What do I get out of splitting themesmanager into its own package?
// The original intention was to split pagemanager into independent modules which can be tested and developed in isolation on their own.
// cryptoutil's KeyBox and PasswordBox are great examples of this
// but kinda feel like themesmanager's goals overlap too tightly with pagemanager's
// At its core themesmanager is about serving a directory of themes and connecting it with a KV store in order to make HTML pages.
// It feels like it could stand on its own and could be useful to someone else, which is why I thought of making it an independent package.

// a template is uniquely identified by its themePath and templateName. Therefore the resultant *template.Template, TemplateVariables, ContentSecurityPolicy can be cached
// Also it needs a KV store
