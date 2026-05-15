// AttachmentValidation — extension allowlist and executable blocklist for notice attachments
package service

import (
	"path/filepath"
	"strings"
)

var blockedAttachmentExts = map[string]struct{}{
	".exe": {}, ".sh": {}, ".bat": {}, ".js": {}, ".php": {},
	".com": {}, ".scr": {}, ".vbs": {}, ".cmd": {}, ".jar": {},
}

var allowedAttachmentExts = map[string]struct{}{
	".pdf": {}, ".hwp": {}, ".hwpx": {},
	".doc": {}, ".docx": {}, ".ppt": {}, ".pptx": {},
	".xls": {}, ".xlsx": {}, ".csv": {}, ".txt": {},
	".zip": {},
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {},
}

// IsBlockedAttachmentExt returns true for executable-like extensions that must be rejected.
func IsBlockedAttachmentExt(ext string) bool {
	_, blocked := blockedAttachmentExts[strings.ToLower(ext)]
	return blocked
}

// IsAllowedAttachmentExt returns true for extensions on the attachment whitelist.
func IsAllowedAttachmentExt(ext string) bool {
	_, ok := allowedAttachmentExts[strings.ToLower(ext)]
	return ok
}

// ExtensionOf returns the lowercase extension (with dot) of a filename.
func ExtensionOf(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}
