package picomatch

// Ported from: picomatch/lib/constants.js

// MaxLength is the maximum allowed input pattern length.
// JS source: constants.js line 89
const MaxLength = 1024 * 64

// --- Posix glob regex constants ---
// JS source: constants.js lines 10-25

const (
	winSlash   = `\\\\/`
	winNoSlash = `[^` + winSlash + `]`
)

const (
	DOT_LITERAL    = `\.`
	PLUS_LITERAL   = `\+`
	QMARK_LITERAL  = `\?`
	SLASH_LITERAL  = `\/`
	ONE_CHAR       = `(?=.)`
	QMARK          = `[^/]`
	END_ANCHOR     = `(?:` + SLASH_LITERAL + `|$)`
	START_ANCHOR   = `(?:^|` + SLASH_LITERAL + `)`
	DOTS_SLASH     = DOT_LITERAL + `{1,2}` + END_ANCHOR
	NO_DOT         = `(?!` + DOT_LITERAL + `)`
	NO_DOTS        = `(?!` + START_ANCHOR + DOTS_SLASH + `)`
	NO_DOT_SLASH   = `(?!` + DOT_LITERAL + `{0,1}` + END_ANCHOR + `)`
	NO_DOTS_SLASH  = `(?!` + DOTS_SLASH + `)`
	QMARK_NO_DOT   = `[^.` + SLASH_LITERAL + `]`
	STAR           = QMARK + `*?`
	SEP            = "/"
)

// PlatformChars holds the regex building blocks for a given platform.
// JS source: constants.js lines 27-65 (POSIX_CHARS / WINDOWS_CHARS)
type PlatformChars struct {
	DOT_LITERAL   string
	PLUS_LITERAL  string
	QMARK_LITERAL string
	SLASH_LITERAL string
	ONE_CHAR      string
	QMARK         string
	END_ANCHOR    string
	DOTS_SLASH    string
	NO_DOT        string
	NO_DOTS       string
	NO_DOT_SLASH  string
	NO_DOTS_SLASH string
	QMARK_NO_DOT  string
	STAR          string
	START_ANCHOR  string
	SEP           string
}

// posixChars are the default POSIX platform characters.
// JS source: constants.js lines 27-44
var posixChars = PlatformChars{
	DOT_LITERAL:   DOT_LITERAL,
	PLUS_LITERAL:  PLUS_LITERAL,
	QMARK_LITERAL: QMARK_LITERAL,
	SLASH_LITERAL: SLASH_LITERAL,
	ONE_CHAR:      ONE_CHAR,
	QMARK:         QMARK,
	END_ANCHOR:    END_ANCHOR,
	DOTS_SLASH:    DOTS_SLASH,
	NO_DOT:        NO_DOT,
	NO_DOTS:       NO_DOTS,
	NO_DOT_SLASH:  NO_DOT_SLASH,
	NO_DOTS_SLASH: NO_DOTS_SLASH,
	QMARK_NO_DOT:  QMARK_NO_DOT,
	STAR:          STAR,
	START_ANCHOR:  START_ANCHOR,
	SEP:           SEP,
}

// windowsChars are the Windows platform characters.
// JS source: constants.js lines 50-65
var windowsChars = PlatformChars{
	DOT_LITERAL:   DOT_LITERAL,
	PLUS_LITERAL:  PLUS_LITERAL,
	QMARK_LITERAL: QMARK_LITERAL,
	SLASH_LITERAL: `[` + winSlash + `]`,
	ONE_CHAR:      ONE_CHAR,
	QMARK:         winNoSlash,
	END_ANCHOR:    `(?:[` + winSlash + `]|$)`,
	DOTS_SLASH:    DOT_LITERAL + `{1,2}(?:[` + winSlash + `]|$)`,
	NO_DOT:        `(?!` + DOT_LITERAL + `)`,
	NO_DOTS:       `(?!(?:^|[` + winSlash + `])` + DOT_LITERAL + `{1,2}(?:[` + winSlash + `]|$))`,
	NO_DOT_SLASH:  `(?!` + DOT_LITERAL + `{0,1}(?:[` + winSlash + `]|$))`,
	NO_DOTS_SLASH: `(?!` + DOT_LITERAL + `{1,2}(?:[` + winSlash + `]|$))`,
	QMARK_NO_DOT:  `[^.` + winSlash + `]`,
	STAR:          winNoSlash + `*?`,
	START_ANCHOR:  `(?:^|[` + winSlash + `])`,
	SEP:           `\`,
}

// GlobChars returns the platform-specific character set.
// JS source: constants.js lines 177-179
func GlobChars(windows bool) PlatformChars {
	if windows {
		return windowsChars
	}
	return posixChars
}

// ExtglobChar describes an extglob operator's regex open/close.
// JS source: constants.js lines 163-171
type ExtglobChar struct {
	Type  string
	Open  string
	Close string
}

// ExtglobChars returns the extglob character definitions for the given platform chars.
// JS source: constants.js lines 163-171
func ExtglobChars(chars PlatformChars) map[byte]ExtglobChar {
	return map[byte]ExtglobChar{
		'!': {Type: "negate", Open: "(?:(?!(?:", Close: "))" + chars.STAR + ")"},
		'?': {Type: "qmark", Open: "(?:", Close: ")?"},
		'+': {Type: "plus", Open: "(?:", Close: ")+"},
		'*': {Type: "star", Open: "(?:", Close: ")*"},
		'@': {Type: "at", Open: "(?:", Close: ")"},
	}
}

// --- POSIX bracket class regex sources ---
// JS source: constants.js lines 71-86
var POSIX_REGEX_SOURCE = map[string]string{
	"alnum":  "a-zA-Z0-9",
	"alpha":  "a-zA-Z",
	"ascii":  `\x00-\x7F`,
	"blank":  ` \t`,
	"cntrl":  `\x00-\x1F\x7F`,
	"digit":  "0-9",
	"graph":  `\x21-\x7E`,
	"lower":  "a-z",
	"print":  `\x20-\x7E `,
	"punct":  `\-!"#$%&'()\*+,./:;<=>?@[\]^_` + "`{|}~",
	"space":  ` \t\r\n\v\f`,
	"upper":  "A-Z",
	"word":   "A-Za-z0-9_",
	"xdigit": "A-Fa-f0-9",
}

// --- Regex constants ---
// JS source: constants.js lines 93-98

// regexBackslash matches backslashes not followed by special regex chars.
// JS: /\\(?![*+?^${}(|)[\]])/g
const regexBackslashPattern = `\\(?![*+?^${}(|)[\]])`

// regexNonSpecialChars matches runs of non-special characters.
// JS: /^[^@![\].,$*+?^{}()|\\/]+/
const regexNonSpecialCharsPattern = `^[^@!\[\].,$*+?^{}()|\\\/]+`

// regexSpecialChars tests if a string contains special regex characters.
// JS: /[-*+?.^${}(|)[\]]/
const regexSpecialCharsPattern = `[-*+?.^${}(|)[\]]`

// regexSpecialCharsBackref captures special chars with optional preceding escape.
// JS: /(\\?)((\W)(\3*))/g
const regexSpecialCharsBackrefPattern = `(\\?)((\W)(\3*))`

// regexSpecialCharsGlobal matches special regex chars for escaping.
// JS: /([-*+?.^${}(|)[\]])/g
const regexSpecialCharsGlobalPattern = `([-*+?.^${}(|)[\]])`

// regexRemoveBackslash matches bracket expressions or escaped chars.
// JS: /(?:\[.*?[^\\]\]|\\(?=.))/g
const regexRemoveBackslashPattern = `(?:\[.*?[^\\]\]|\\(?=.))`

// --- Replacements for reducing parsing time ---
// JS source: constants.js lines 101-106
var REPLACEMENTS = map[string]string{
	"***":     "*",
	"**/**":   "**",
	"**/**/**": "**",
}

// --- Character code constants ---
// JS source: constants.js lines 109-157
const (
	CHAR_0 = 48 // 0
	CHAR_9 = 57 // 9

	CHAR_UPPERCASE_A = 65  // A
	CHAR_LOWERCASE_A = 97  // a
	CHAR_UPPERCASE_Z = 90  // Z
	CHAR_LOWERCASE_Z = 122 // z

	CHAR_LEFT_PARENTHESES  = 40 // (
	CHAR_RIGHT_PARENTHESES = 41 // )
	CHAR_ASTERISK          = 42 // *

	CHAR_AMPERSAND          = 38  // &
	CHAR_AT                 = 64  // @
	CHAR_BACKWARD_SLASH     = 92  // \
	CHAR_CARRIAGE_RETURN    = 13  // \r
	CHAR_CIRCUMFLEX_ACCENT  = 94  // ^
	CHAR_COLON              = 58  // :
	CHAR_COMMA              = 44  // ,
	CHAR_DOT                = 46  // .
	CHAR_DOUBLE_QUOTE       = 34  // "
	CHAR_EQUAL              = 61  // =
	CHAR_EXCLAMATION_MARK   = 33  // !
	CHAR_FORM_FEED          = 12  // \f
	CHAR_FORWARD_SLASH      = 47  // /
	CHAR_GRAVE_ACCENT       = 96  // `
	CHAR_HASH               = 35  // #
	CHAR_HYPHEN_MINUS       = 45  // -
	CHAR_LEFT_ANGLE_BRACKET = 60  // <
	CHAR_LEFT_CURLY_BRACE   = 123 // {
	CHAR_LEFT_SQUARE_BRACKET  = 91  // [
	CHAR_LINE_FEED            = 10  // \n
	CHAR_NO_BREAK_SPACE       = 160 // \u00A0
	CHAR_PERCENT              = 37  // %
	CHAR_PLUS                 = 43  // +
	CHAR_QUESTION_MARK        = 63  // ?
	CHAR_RIGHT_ANGLE_BRACKET  = 62  // >
	CHAR_RIGHT_CURLY_BRACE    = 125 // }
	CHAR_RIGHT_SQUARE_BRACKET = 93  // ]
	CHAR_SEMICOLON            = 59  // ;
	CHAR_SINGLE_QUOTE         = 39  // '
	CHAR_SPACE                = 32  //
	CHAR_TAB                  = 9   // \t
	CHAR_UNDERSCORE           = 95  // _
	CHAR_VERTICAL_LINE        = 124 // |
	CHAR_ZERO_WIDTH_NOBREAK_SPACE = 65279 // \uFEFF
)
