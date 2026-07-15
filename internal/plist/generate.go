// Package plist renders launchd plist XML for an envonce-managed service.
package plist

import (
	"strconv"
	"strings"
)

// PlistData holds the fields rendered into a launchd plist.
type PlistData struct {
	Label            string
	WrapperPath      string
	RunAtLoad        bool
	KeepAlive        bool
	ThrottleInterval int
	StdoutPath       string
	StderrPath       string
}

// xmlEscape escapes the XML characters that may appear in plist string values:
// &, <, >, and ".
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&#34;")
	return s
}

// boolTag renders a <key>/<true|false/> pair with the same indentation as
// the surrounding string key entries.
func boolTag(name string, v bool) string {
	val := "false"
	if v {
		val = "true"
	}
	return "  <key>" + name + "</key><" + val + "/>"
}

// Generate produces a launchd plist document from d. The returned string
// always ends with a trailing newline.
func Generate(d PlistData) (string, error) {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	b.WriteString("  <key>Label</key><string>" + xmlEscape(d.Label) + "</string>\n")
	b.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	b.WriteString("    <string>" + xmlEscape(d.WrapperPath) + "</string>\n")
	b.WriteString("  </array>\n")
	b.WriteString(boolTag("RunAtLoad", d.RunAtLoad) + "\n")
	b.WriteString(boolTag("KeepAlive", d.KeepAlive) + "\n")
	b.WriteString("  <key>ThrottleInterval</key><integer>" + strconv.Itoa(d.ThrottleInterval) + "</integer>\n")
	b.WriteString("  <key>StandardOutPath</key><string>" + xmlEscape(d.StdoutPath) + "</string>\n")
	b.WriteString("  <key>StandardErrorPath</key><string>" + xmlEscape(d.StderrPath) + "</string>\n")
	b.WriteString("</dict>\n</plist>\n")
	return b.String(), nil
}
