package hy

import "strings"

const sanitizerFailsafe = "ZhypergoZ"

var allowedTags = map[string]struct{}{
	"html": {}, "base": {}, "link": {}, "meta": {} /*, "style": {}*/, "title": {}, "body": {}, "address": {},
	"article": {}, "aside": {}, "footer": {}, "header": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"main": {}, "nav": {}, "section": {}, "blockquote": {}, "dd": {}, "div": {}, "dl": {}, "dt": {}, "figcaption": {},
	"figure": {}, "hr": {}, "li": {}, "ol": {}, "p": {}, "pre": {}, "ul": {}, "a": {}, "abbr": {}, "b": {}, "bdi": {},
	"bdo": {}, "br": {}, "cite": {}, "code": {}, "data": {}, "dfn": {}, "em": {}, "i": {}, "kbd": {}, "mark": {},
	"q": {}, "rp": {}, "rt": {}, "ruby": {}, "s": {}, "samp": {}, "small": {}, "span": {}, "strong": {}, "sub": {},
	"sup": {}, "time": {}, "u": {}, "var": {}, "wbr": {}, "area": {}, "audio": {}, "img": {}, "map": {}, "track": {},
	"video": {}, "embed": {} /*, "iframe": {}*/, "object": {}, "param": {}, "picture": {}, "portal": {}, "source": {},
	"svg": {}, "math": {}, "canvas": {}, "noscript": {} /*, "script": {}*/, "del": {}, "ins": {}, "caption": {},
	"col": {}, "colgroup": {}, "table": {}, "tbody": {}, "td": {}, "tfoot": {}, "th": {}, "thead": {}, "tr": {},
	"button": {}, "datalist": {}, "fieldset": {}, "form": {}, "input": {}, "label": {}, "legend": {}, "meter": {},
	"optgroup": {}, "option": {}, "output": {}, "progress": {}, "select": {}, "textarea": {}, "details": {},
	"dialog": {}, "menu": {}, "summary": {}, "slot": {}, "template": {},
}

var globalAttributes = map[string]struct{}{
	"accesskey": {}, "autocapitalize": {}, "class": {}, "contenteditable": {}, "dir": {}, "draggable": {},
	"enterkeyhint": {}, "hidden": {}, "id": {}, "inputmode": {}, "is": {}, "itemid": {}, "itemprop": {},
	"itemref": {}, "itemscope": {}, "itemtype": {}, "lang": {}, "nonce": {}, "part": {}, "slot": {},
	"spellcheck": {} /*, "style": {}*/, "tabindex": {}, "title": {}, "translate": {},
}

var tagAttributes = map[string]map[string]struct{}{
	"html": {"xmlns": {}},
	"base": {"href": {}, "target": {}},
	"link": {
		"as": {}, "crossorigin": {}, "disabled": {}, "href": {}, "hreflang": {}, "imagesizes": {}, "imagesrcset": {},
		"integrity": {}, "media": {}, "prefetch": {}, "referrerpolicy": {}, "rel": {}, "sizes": {}, "title": {}, "type": {},
	},
	"meta":       {"charset": {}, "content": {}, "http-equiv": {}, "name": {}},
	"style":      {"type": {}, "media": {}, "nonce": {}, "title": {}},
	"blockquote": {"cite": {}},
	"hr":         {"color": {}},
	"li":         {"value": {}},
	"ol":         {"reversed": {}, "start": {}, "type": {}},
	"a":          {"download": {}, "href": {}, "hreflang": {}, "ping": {}, "referrerpolicy": {}, "rel": {}, "target": {}, "type": {}},
	"data":       {"value": {}},
	"q":          {"cite": {}},
	"time":       {"datetime": {}},
	"area": {
		"alt": {}, "coords": {}, "download": {}, "href": {}, "hreflang": {}, "ping": {}, "referrerpolicy": {},
		"rel": {}, "shape": {}, "target": {},
	},
	"audio": {
		"autoplay": {}, "controls": {}, "crossorigin": {}, "currentTime": {}, "disableRemotePlayback": {},
		"loop": {}, "muted": {}, "preload": {}, "src": {},
	},
	"img": {
		"alt": {}, "crossorigin": {}, "decoding": {}, "height": {}, "ismap": {}, "loading": {}, "referrerpolicy": {},
		"sizes": {}, "src": {}, "srcset": {}, "width": {}, "usemap": {},
	},
	"map":   {"name": {}},
	"track": {"default": {}, "kind": {}, "label": {}, "src": {}, "srclang": {}},
	"video": {
		"autoplay": {}, "autoPictureInPicture": {}, "buffered": {}, "controls": {}, "controlslist": {},
		"crossorigin": {}, "currentTime": {}, "disabledPictureInPicture": {}, "disableRemotePlayback": {}, "height": {},
		"loop": {}, "muted": {}, "playsinline": {}, "poster": {}, "preload": {}, "src": {}, "width": {},
	},
	"embed": {"height": {}, "src": {}, "type": {}, "width": {}},
	"iframe": {
		"allow": {}, "allowfullscreen": {}, "allowpaymentrequest": {}, "csp": {}, "height": {}, "loading": {},
		"name": {}, "referrerpolicy": {}, "sandbox": {}, "src": {}, "srcdoc": {}, "width": {},
	},
	"object": {"data": {}, "form": {}, "height": {}, "name": {}, "type": {}, "usemap": {}, "width": {}},
	"param":  {"name": {}, "value": {}},
	"portal": {"referrerpolicy": {}, "src": {}},
	"source": {"media": {}, "sizes": {}, "src": {}, "srcset": {}, "type": {}},
	"canvas": {"height": {}, "width": {}},
	"script": {
		"async": {}, "crossorigin": {}, "defer": {}, "integrity": {}, "nomodule": {}, "nonce": {}, "referrerpolicy": {},
		"src": {}, "type": {},
	},
	"del":      {"cite": {}, "datetime": {}},
	"ins":      {"cite": {}, "datetime": {}},
	"col":      {"span": {}},
	"colgroup": {"span": {}},
	"td":       {"colspan": {}, "headers": {}, "rowspan": {}},
	"th":       {"abbr": {}, "colspan": {}, "headers": {}, "rowspan": {}, "scope": {}},
	"button": {
		"autofocus": {}, "disabled": {}, "form": {}, "formaction": {}, "formenctype": {}, "formmethod": {},
		"formnovalidate": {}, "formtarget": {}, "name": {}, "type": {}, "value": {},
	},
	"fieldset": {"disabled": {}, "form": {}, "name": {}},
	"form": {
		"accept-charset": {}, "autocomplete": {}, "name": {}, "rel": {}, "action": {}, "enctype": {}, "method": {},
		"novalidate": {}, "target": {},
	},
	"input": {
		"accept": {}, "alt": {}, "autocomplete": {}, "autofocus": {}, "capture": {}, "checked": {}, "dirname": {},
		"disabled": {}, "form": {}, "formaction": {}, "formenctype": {}, "formmethod": {}, "formnovalidate": {},
		"formtarget": {}, "height": {}, "list": {}, "max": {}, "maxlength": {}, "min": {}, "minlength": {},
		"multiple": {}, "name": {}, "pattern": {}, "placeholder": {}, "readonly": {}, "required": {}, "size": {},
		"src": {}, "step": {}, "type": {}, "value": {}, "width": {},
	},
	"label":    {"for": {}},
	"meter":    {"value": {}, "min": {}, "max": {}, "low": {}, "high": {}, "optimum": {}, "form": {}},
	"optgroup": {"disabled": {}, "label": {}},
	"option":   {"disabled": {}, "label": {}, "selected": {}, "value": {}},
	"output":   {"for": {}, "form": {}, "name": {}},
	"progress": {"max": {}, "value": {}},
	"select": {
		"autocomplete": {}, "autofocus": {}, "disabled": {}, "form": {}, "multiple": {}, "name": {},
		"required": {}, "size": {},
	},
	"textarea": {
		"autocomplete": {}, "autofocus": {}, "cols": {}, "disabled": {}, "form": {}, "maxlength": {}, "name": {},
		"placeholder": {}, "readonly": {}, "required": {}, "rows": {}, "spellcheck": {}, "wrap": {},
	},
	"details": {"open": {}},
	"dialog":  {"open": {}},
	"menu":    {"type": {}},
	"slot":    {"name": {}},
}

func isInSet(s string, sets ...map[string]struct{}) bool {
	for _, set := range sets {
		if _, ok := set[s]; ok {
			return true
		}
	}
	return false
}

func DefaultSanitizer(tag string, attrName string, attrValue string) bool {
	tag = strings.ToLower(tag)
	if exists := isInSet(tag, allowedTags); !exists {
		return isInSet(attrName, globalAttributes)
	} else if attrName == "" {
		return exists
	}
	attrName = strings.ToLower(attrName)
	if isURLAttr(attrName) {
		return isSafeURL(attrValue)
	}
	return isInSet(attrName, globalAttributes, tagAttributes[tag]) ||
		strings.HasPrefix(attrName, "data-") ||
		tag == "svg" ||
		tag == "math"
}

func AllowTags(tags ...string) Sanitizer {
	userAllowedTags := make(map[string]struct{})
	for _, tag := range tags {
		userAllowedTags[tag] = struct{}{}
	}
	return func(tag string, attrName string, attrValue string) bool {
		tag = strings.ToLower(tag)
		if exists := isInSet(tag, userAllowedTags, allowedTags); !exists {
			return isInSet(attrName, globalAttributes)
		} else if attrName == "" {
			return exists
		}
		attrName = strings.ToLower(attrName)
		if isURLAttr(attrName) {
			return isSafeURL(attrValue)
		}
		return isInSet(attrName, globalAttributes, tagAttributes[tag]) ||
			strings.HasPrefix(attrName, "data-") ||
			tag == "svg" ||
			tag == "math"
	}
}

var urlAttrNames = map[string]struct{}{
	"action": {}, "archive": {}, "background": {}, "cite": {}, "classid": {}, "codebase": {}, "data": {},
	"formaction": {}, "href": {}, "icon": {}, "longdesc": {}, "manifest": {}, "poster": {}, "profile": {}, "src": {},
	"usemap": {}, "xmlns": {},
}

// html/template.attrType
func isURLAttr(name string) bool {
	if strings.HasPrefix(name, "data-") {
		// Strip data- so that custom attribute heuristics below are
		// widely applied.
		// Treat data-action as URL below.
		name = name[5:]
	} else if colon := strings.IndexRune(name, ':'); colon != -1 {
		if name[:colon] == "xmlns" {
			return true
		}
		// Treat svg:href and xlink:href as href below.
		name = name[colon+1:]
	}
	if _, ok := urlAttrNames[name]; ok {
		return true
	}
	// Heuristics to prevent "javascript:..." injection in custom
	// data attributes and custom attributes like g:tweetUrl.
	// https://www.w3.org/TR/html5/dom.html#embedding-custom-non-visible-data-with-the-data-*-attributes
	// "Custom data attributes are intended to store custom data
	//  private to the page or application, for which there are no
	//  more appropriate attributes or elements."
	// Developers seem to store URL content in data URLs that start
	// or end with "URI" or "URL".
	if strings.Contains(name, "src") ||
		strings.Contains(name, "uri") ||
		strings.Contains(name, "url") {
		return true
	}
	return false
}

// html/template.isSafeURL
func isSafeURL(s string) bool {
	if i := strings.IndexRune(s, ':'); i >= 0 && !strings.ContainsRune(s[:i], '/') {
		protocol := s[:i]
		if !strings.EqualFold(protocol, "http") && !strings.EqualFold(protocol, "https") && !strings.EqualFold(protocol, "mailto") {
			return false
		}
	}
	return true
}
