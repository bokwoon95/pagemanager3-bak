package hy

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

var htmlReplacementTable = []string{
	0:    "\uFFFD",
	'"':  "&#34;",
	'&':  "&amp;",
	'\'': "&#39;",
	'+':  "&#43;",
	'<':  "&lt;",
	'>':  "&gt;",
}

var attrNameReplacementTable = []string{
	0:    "&#xfffd;",
	'\t': "&#9;",
	'\n': "&#10;",
	'\v': "&#11;",
	'\f': "&#12;",
	'\r': "&#13;",
	' ':  "&#32;",
	'"':  "&#34;",
	'&':  "&amp;",
	'\'': "&#39;",
	'+':  "&#43;",
	'<':  "&lt;",
	'=':  "&#61;",
	'>':  "&gt;",
	// A parse error in the attribute value (unquoted) and
	// before attribute value states.
	// Treated as a quoting character by IE.
	'`': "&#96;",
}

// html/template.htmlReplacer
func escapeHTML(buf *bytes.Buffer, replacementTable []string, s string) {
	r, w, written := rune(0), 0, 0
	for i := 0; i < len(s); i += w {
		// Cannot use 'for range s' because we need to preserve the width
		// of the runes in the input. If we see a decoding error, the input
		// width will not be utf8.Runelen(r) and we will overrun the buffer.
		r, w = utf8.DecodeRuneInString(s[i:])
		if int(r) < len(replacementTable) {
			if repl := replacementTable[r]; len(repl) != 0 {
				if written == 0 {
					buf.Grow(len(s))
				}
				buf.WriteString(s[written:i])
				buf.WriteString(repl)
				written = i + w
			}
		}
	}
	if written == 0 {
		buf.WriteString(s)
		return
	}
	buf.WriteString(s[written:])
}

// html/template.isHex
func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// html/template.processURLOnto
func escapeURL(buf *bytes.Buffer, norm bool, s string) {
	buf.Grow(len(s) + 16)
	written := 0
	// The byte loop below assumes that all URLs use UTF-8 as the
	// content-encoding. This is similar to the URI to IRI encoding scheme
	// defined in section 3.1 of  RFC 3987, and behaves the same as the
	// EcmaScript builtin encodeURIComponent.
	// It should not cause any misencoding of URLs in pages with
	// Content-type: text/html;charset=UTF-8.
	for i, n := 0, len(s); i < n; i++ {
		c := s[i]
		switch c {
		// Single quote and parens are sub-delims in RFC 3986, but we
		// escape them so the output can be embedded in single
		// quoted attributes and unquoted CSS url(...) constructs.
		// Single quotes are reserved in URLs, but are only used in
		// the obsolete "mark" rule in an appendix in RFC 3986
		// so can be safely encoded.
		case '!', '#', '$', '&', '*', '+', ',', '/', ':', ';', '=', '?', '@', '[', ']':
			if norm {
				continue
			}
		// Unreserved according to RFC 3986 sec 2.3
		// "For consistency, percent-encoded octets in the ranges of
		// ALPHA (%41-%5A and %61-%7A), DIGIT (%30-%39), hyphen (%2D),
		// period (%2E), underscore (%5F), or tilde (%7E) should not be
		// created by URI producers
		case '-', '.', '_', '~':
			continue
		case '%':
			// When normalizing do not re-encode valid escapes.
			if norm && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
				continue
			}
		default:
			// Unreserved according to RFC 3986 sec 2.3
			if 'a' <= c && c <= 'z' {
				continue
			}
			if 'A' <= c && c <= 'Z' {
				continue
			}
			if '0' <= c && c <= '9' {
				continue
			}
		}
		buf.WriteString(s[written:i])
		fmt.Fprintf(buf, "%%%02x", c)
		written = i + 1
	}
	buf.WriteString(s[written:])
}

// html/template.htmlSpaceAndASCIIAlnumBytes
const htmlSpaceAndASCIIAlnumBytes = "\x00\x36\x00\x00\x01\x00\xff\x03\xfe\xff\xff\x07\xfe\xff\xff\x07"

// html/template.isHTMLSpace
func isHTMLSpace(c byte) bool {
	return (c <= 0x20) && 0 != (htmlSpaceAndASCIIAlnumBytes[c>>3]&(1<<uint(c&0x7)))
}

// html/template.isHTMLSpaceOrASCIIAlnum
func isHTMLSpaceOrASCIIAlnum(c byte) bool {
	return (c < 0x80) && 0 != (htmlSpaceAndASCIIAlnumBytes[c>>3]&(1<<uint(c&0x7)))
}

// html/template.filterSrcsetElement
func sanitizeAndEscapeSrcsetElement(buf *bytes.Buffer, s string, left int, right int) {
	start := left
	for start < right && isHTMLSpace(s[start]) {
		start++
	}
	end := right
	for i := start; i < right; i++ {
		if isHTMLSpace(s[i]) {
			end = i
			break
		}
	}
	if url := s[start:end]; isSafeURL(url) {
		// If image metadata is only spaces or alnums then
		// we don't need to URL normalize it.
		metadataOk := true
		for i := end; i < right; i++ {
			if !isHTMLSpaceOrASCIIAlnum(s[i]) {
				metadataOk = false
				break
			}
		}
		if metadataOk {
			buf.WriteString(s[left:start])
			escapeURL(buf, true, url)
			buf.WriteString(s[end:right])
			return
		}
	}
	buf.WriteString("#")
	buf.WriteString(sanitizerFailsafe)
}

// html/template.srcsetFilterAndEscaper
func sanitizeAndEscapeSrcset(buf *bytes.Buffer, s string) {
	written := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			sanitizeAndEscapeSrcsetElement(buf, s, written, i)
			buf.WriteString(",")
			written = i + 1
		}
	}
	sanitizeAndEscapeSrcsetElement(buf, s, written, len(s))
}
