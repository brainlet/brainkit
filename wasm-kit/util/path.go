package util

import (
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/common"
)

const separator = CharCodeSlash

func NormalizePath(path string) string {
	pos := 0
	length := len(path)

	// trim leading './'
	for pos+1 < length &&
		path[pos] == byte(CharCodeDot) &&
		path[pos+1] == byte(separator) {
		pos += 2
	}

	if pos > 0 {
		path = path[pos:]
		length -= pos
		pos = 0
	}

	for pos+1 < length {
		atEnd := false

		// we are only interested in '/.' sequences ...
		if path[pos] == byte(separator) &&
			path[pos+1] == byte(CharCodeDot) {

			// '/.' ( '/' | $ )
			atEnd = pos+2 == length
			if atEnd ||
				(pos+2 < length && path[pos+2] == byte(separator)) {
				if atEnd {
					path = path[:pos]
				} else {
					path = path[:pos] + path[pos+2:]
				}
				length -= 2
				continue
			}

			// '/.' ( './' | '.' $ )
			atEnd = pos+3 == length
			if (atEnd && path[pos+2] == byte(CharCodeDot)) ||
				(pos+3 < length &&
					path[pos+2] == byte(CharCodeDot) &&
					path[pos+3] == byte(separator)) {

				// find preceding '/'
				ipos := pos
				ipos--
				found := false
				for ipos >= 0 {
					if path[ipos] == byte(separator) {
						if pos-ipos != 3 ||
							path[ipos+1] != byte(CharCodeDot) ||
							path[ipos+2] != byte(CharCodeDot) {
							// exclude '..' itself
							if atEnd {
								path = path[:ipos]
							} else {
								path = path[:ipos] + path[pos+3:]
							}
							length -= pos + 3 - ipos
							pos = ipos - 1 // incremented again at end of loop
						}
						found = true
						break
					}
					ipos--
				}

				// if there's no preceding '/', trim start if non-empty
				if !found && pos > 0 {
					if pos != 2 ||
						path[0] != byte(CharCodeDot) ||
						path[1] != byte(CharCodeDot) {
						// exclude '..' itself
						path = path[pos+4:]
						length = len(path)
						continue
					}
				}
			}
		}
		pos++
	}

	if length > 0 {
		return path
	}
	return "."
}

func ResolvePath(normalizedPath string, origin string) string {
	if strings.HasPrefix(normalizedPath, "std/") {
		return normalizedPath
	}
	return NormalizePath(
		Dirname(origin) + common.PATH_DELIMITER + normalizedPath,
	)
}

func Dirname(normalizedPath string) string {
	pos := len(normalizedPath)
	if pos <= 1 {
		if pos == 0 {
			return "."
		}
		if normalizedPath[0] == byte(separator) {
			return normalizedPath
		}
	}
	pos--
	for pos > 0 {
		if normalizedPath[pos] == byte(separator) {
			return normalizedPath[:pos]
		}
		pos--
	}
	return "."
}
