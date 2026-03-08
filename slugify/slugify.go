package slugify

// Go port of slugify
// TS source: https://github.com/simov/slugify/blob/master/slugify.js

import (
	"regexp"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// String returns a pointer to the given string value.
// Convenience helper for use with option structs.
func String(v string) *string { return &v }

// Bool returns a pointer to the given bool value.
// Convenience helper for use with option structs.
func Bool(v bool) *bool { return &v }

// ---------------------------------------------------------------------------
// Type definitions
// TS source: slugify.d.ts
// ---------------------------------------------------------------------------

// Options configures the slugify behavior.
// All fields are optional — nil means "use default".
// TS source: slugify.d.ts lines 12-19
type Options struct {
	Replacement *string        // replace spaces with this character, default: "-"
	Remove      *regexp.Regexp // remove characters matching this regex, default: /[^\w\s$*_+~.()'"!\-:@]+/g
	Lower       *bool          // convert to lower case, default: false
	Strict      *bool          // strip special characters except replacement, default: false
	Locale      *string        // language code of the locale to use
	Trim        *bool          // trim leading and trailing replacement chars, default: true
}

// ---------------------------------------------------------------------------
// Default remove regex
// TS source: line 42 — /[^\w\s$*_+~.()'"!\-:@]+/g
// ---------------------------------------------------------------------------

var defaultRemoveRegex = regexp.MustCompile(`[^\w\s$*_+~.()'\"!:@-]+`)

// Strict mode regex: keep only A-Za-z0-9 and whitespace.
// TS source: line 46 — /[^A-Za-z0-9\s]/g
var strictRegex = regexp.MustCompile(`[^A-Za-z0-9\s]`)

// Whitespace sequence regex for collapsing spaces into replacement.
// TS source: line 55 — /\s+/g
var whitespaceRegex = regexp.MustCompile(`\s+`)

// ---------------------------------------------------------------------------
// Global state
// TS source: line 15 (charMap), line 16 (locales)
// ---------------------------------------------------------------------------

var (
	mu sync.RWMutex
)

// ---------------------------------------------------------------------------
// Slugify
// TS source: lines 18-62
// ---------------------------------------------------------------------------

// Slugify converts a string into a URL-friendly slug.
//
// The function transliterates Unicode characters to their ASCII equivalents
// using a built-in character map, applies locale-specific overrides when
// specified, and normalizes whitespace into a replacement character.
//
//	slugify.Slugify("foo bar")                                    // "foo-bar"
//	slugify.Slugify("foo bar", slugify.Options{Lower: Bool(true)}) // "foo-bar"
//	slugify.Slugify("Ä Ö Ü", slugify.Options{Locale: String("de")}) // "AE-OE-UE"
func Slugify(s string, opts ...Options) string {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Resolve defaults.
	// TS source: line 29 — replacement defaults to "-"
	replacement := "-"
	if opt.Replacement != nil {
		replacement = *opt.Replacement
	}

	// TS source: line 31 — trim defaults to true
	trim := true
	if opt.Trim != nil {
		trim = *opt.Trim
	}

	// TS source: line 27 — locale lookup
	mu.RLock()
	localMap := map[string]string(nil)
	if opt.Locale != nil {
		localMap = localeMap[*opt.Locale]
	}
	currentCharMap := charMap
	mu.RUnlock()

	// Determine the remove regex.
	// TS source: line 42 — options.remove || default regex
	removeRe := defaultRemoveRegex
	if opt.Remove != nil {
		removeRe = opt.Remove
	}

	// TS source: lines 33-43
	// Build the slug by iterating over each rune.
	// Note: TS does string.normalize() (NFC) first. Go strings are already
	// UTF-8 encoded; full NFC normalization would require golang.org/x/text
	// which is outside stdlib. The charMap handles the common precomposed forms.
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		ch := string(r)

		// TS source: line 36-38 — locale → charMap → original
		appendStr := ""
		found := false

		if localMap != nil {
			if v, ok := localMap[ch]; ok {
				appendStr = v
				found = true
			}
		}
		if !found {
			if v, ok := currentCharMap[ch]; ok {
				appendStr = v
				found = true
			}
		}
		if !found {
			appendStr = ch
		}

		// TS source: line 39 — if mapped char equals replacement, use space
		if appendStr == replacement {
			appendStr = " "
		}

		// TS source: line 42 — apply remove regex to the mapped string
		appendStr = removeRe.ReplaceAllString(appendStr, "")

		b.WriteString(appendStr)
	}

	slug := b.String()

	// TS source: lines 45-47 — strict mode strips non-alphanumeric
	if opt.Strict != nil && *opt.Strict {
		slug = strictRegex.ReplaceAllString(slug, "")
	}

	// TS source: lines 49-51 — trim whitespace
	if trim {
		slug = strings.TrimSpace(slug)
	}

	// TS source: line 55 — replace whitespace sequences with replacement
	slug = whitespaceRegex.ReplaceAllString(slug, replacement)

	// TS source: lines 57-59 — lowercase
	if opt.Lower != nil && *opt.Lower {
		slug = strings.ToLower(slug)
	}

	return slug
}

// ---------------------------------------------------------------------------
// Extend
// TS source: lines 64-66
// ---------------------------------------------------------------------------

// Extend adds or overrides entries in the global character map.
// This affects all subsequent calls to Slugify across the entire process.
//
//	slugify.Extend(map[string]string{"☢": "radioactive"})
//	slugify.Slugify("unicode ♥ is ☢") // "unicode-love-is-radioactive"
func Extend(customMap map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	for k, v := range customMap {
		charMap[k] = v
	}
}


// ---------------------------------------------------------------------------
// Character map
// TS source: line 15 (charMap) — from config/charmap.json
// ---------------------------------------------------------------------------

var charMap = map[string]string{
	"$": "dollar",
	"%": "percent",
	"&": "and",
	"<": "less",
	">": "greater",
	"|": "or",
	"\u00a2": "cent",
	"\u00a3": "pound",
	"\u00a4": "currency",
	"\u00a5": "yen",
	"\u00a9": "(c)",
	"\u00aa": "a",
	"\u00ae": "(r)",
	"\u00ba": "o",
	"\u00c0": "A",
	"\u00c1": "A",
	"\u00c2": "A",
	"\u00c3": "A",
	"\u00c4": "A",
	"\u00c5": "A",
	"\u00c6": "AE",
	"\u00c7": "C",
	"\u00c8": "E",
	"\u00c9": "E",
	"\u00ca": "E",
	"\u00cb": "E",
	"\u00cc": "I",
	"\u00cd": "I",
	"\u00ce": "I",
	"\u00cf": "I",
	"\u00d0": "D",
	"\u00d1": "N",
	"\u00d2": "O",
	"\u00d3": "O",
	"\u00d4": "O",
	"\u00d5": "O",
	"\u00d6": "O",
	"\u00d8": "O",
	"\u00d9": "U",
	"\u00da": "U",
	"\u00db": "U",
	"\u00dc": "U",
	"\u00dd": "Y",
	"\u00de": "TH",
	"\u00df": "ss",
	"\u00e0": "a",
	"\u00e1": "a",
	"\u00e2": "a",
	"\u00e3": "a",
	"\u00e4": "a",
	"\u00e5": "a",
	"\u00e6": "ae",
	"\u00e7": "c",
	"\u00e8": "e",
	"\u00e9": "e",
	"\u00ea": "e",
	"\u00eb": "e",
	"\u00ec": "i",
	"\u00ed": "i",
	"\u00ee": "i",
	"\u00ef": "i",
	"\u00f0": "d",
	"\u00f1": "n",
	"\u00f2": "o",
	"\u00f3": "o",
	"\u00f4": "o",
	"\u00f5": "o",
	"\u00f6": "o",
	"\u00f8": "o",
	"\u00f9": "u",
	"\u00fa": "u",
	"\u00fb": "u",
	"\u00fc": "u",
	"\u00fd": "y",
	"\u00fe": "th",
	"\u00ff": "y",
	"\u0100": "A",
	"\u0101": "a",
	"\u0102": "A",
	"\u0103": "a",
	"\u0104": "A",
	"\u0105": "a",
	"\u0106": "C",
	"\u0107": "c",
	"\u010c": "C",
	"\u010d": "c",
	"\u010e": "D",
	"\u010f": "d",
	"\u0110": "DJ",
	"\u0111": "dj",
	"\u0112": "E",
	"\u0113": "e",
	"\u0116": "E",
	"\u0117": "e",
	"\u0118": "e",
	"\u0119": "e",
	"\u011a": "E",
	"\u011b": "e",
	"\u011e": "G",
	"\u011f": "g",
	"\u0122": "G",
	"\u0123": "g",
	"\u0128": "I",
	"\u0129": "i",
	"\u012a": "i",
	"\u012b": "i",
	"\u012e": "I",
	"\u012f": "i",
	"\u0130": "I",
	"\u0131": "i",
	"\u0136": "k",
	"\u0137": "k",
	"\u013b": "L",
	"\u013c": "l",
	"\u013d": "L",
	"\u013e": "l",
	"\u0141": "L",
	"\u0142": "l",
	"\u0143": "N",
	"\u0144": "n",
	"\u0145": "N",
	"\u0146": "n",
	"\u0147": "N",
	"\u0148": "n",
	"\u014c": "O",
	"\u014d": "o",
	"\u0150": "O",
	"\u0151": "o",
	"\u0152": "OE",
	"\u0153": "oe",
	"\u0154": "R",
	"\u0155": "r",
	"\u0158": "R",
	"\u0159": "r",
	"\u015a": "S",
	"\u015b": "s",
	"\u015e": "S",
	"\u015f": "s",
	"\u0160": "S",
	"\u0161": "s",
	"\u0162": "T",
	"\u0163": "t",
	"\u0164": "T",
	"\u0165": "t",
	"\u0168": "U",
	"\u0169": "u",
	"\u016a": "u",
	"\u016b": "u",
	"\u016e": "U",
	"\u016f": "u",
	"\u0170": "U",
	"\u0171": "u",
	"\u0172": "U",
	"\u0173": "u",
	"\u0174": "W",
	"\u0175": "w",
	"\u0176": "Y",
	"\u0177": "y",
	"\u0178": "Y",
	"\u0179": "Z",
	"\u017a": "z",
	"\u017b": "Z",
	"\u017c": "z",
	"\u017d": "Z",
	"\u017e": "z",
	"\u018f": "E",
	"\u0192": "f",
	"\u01a0": "O",
	"\u01a1": "o",
	"\u01af": "U",
	"\u01b0": "u",
	"\u01c8": "LJ",
	"\u01c9": "lj",
	"\u01cb": "NJ",
	"\u01cc": "nj",
	"\u0218": "S",
	"\u0219": "s",
	"\u021a": "T",
	"\u021b": "t",
	"\u0259": "e",
	"\u02da": "o",
	"\u0386": "A",
	"\u0388": "E",
	"\u0389": "H",
	"\u038a": "I",
	"\u038c": "O",
	"\u038e": "Y",
	"\u038f": "W",
	"\u0390": "i",
	"\u0391": "A",
	"\u0392": "B",
	"\u0393": "G",
	"\u0394": "D",
	"\u0395": "E",
	"\u0396": "Z",
	"\u0397": "H",
	"\u0398": "8",
	"\u0399": "I",
	"\u039a": "K",
	"\u039b": "L",
	"\u039c": "M",
	"\u039d": "N",
	"\u039e": "3",
	"\u039f": "O",
	"\u03a0": "P",
	"\u03a1": "R",
	"\u03a3": "S",
	"\u03a4": "T",
	"\u03a5": "Y",
	"\u03a6": "F",
	"\u03a7": "X",
	"\u03a8": "PS",
	"\u03a9": "W",
	"\u03aa": "I",
	"\u03ab": "Y",
	"\u03ac": "a",
	"\u03ad": "e",
	"\u03ae": "h",
	"\u03af": "i",
	"\u03b0": "y",
	"\u03b1": "a",
	"\u03b2": "b",
	"\u03b3": "g",
	"\u03b4": "d",
	"\u03b5": "e",
	"\u03b6": "z",
	"\u03b7": "h",
	"\u03b8": "8",
	"\u03b9": "i",
	"\u03ba": "k",
	"\u03bb": "l",
	"\u03bc": "m",
	"\u03bd": "n",
	"\u03be": "3",
	"\u03bf": "o",
	"\u03c0": "p",
	"\u03c1": "r",
	"\u03c2": "s",
	"\u03c3": "s",
	"\u03c4": "t",
	"\u03c5": "y",
	"\u03c6": "f",
	"\u03c7": "x",
	"\u03c8": "ps",
	"\u03c9": "w",
	"\u03ca": "i",
	"\u03cb": "y",
	"\u03cc": "o",
	"\u03cd": "y",
	"\u03ce": "w",
	"\u0401": "Yo",
	"\u0402": "DJ",
	"\u0404": "Ye",
	"\u0406": "I",
	"\u0407": "Yi",
	"\u0408": "J",
	"\u0409": "LJ",
	"\u040a": "NJ",
	"\u040b": "C",
	"\u040f": "DZ",
	"\u0410": "A",
	"\u0411": "B",
	"\u0412": "V",
	"\u0413": "G",
	"\u0414": "D",
	"\u0415": "E",
	"\u0416": "Zh",
	"\u0417": "Z",
	"\u0418": "I",
	"\u0419": "J",
	"\u041a": "K",
	"\u041b": "L",
	"\u041c": "M",
	"\u041d": "N",
	"\u041e": "O",
	"\u041f": "P",
	"\u0420": "R",
	"\u0421": "S",
	"\u0422": "T",
	"\u0423": "U",
	"\u0424": "F",
	"\u0425": "H",
	"\u0426": "C",
	"\u0427": "Ch",
	"\u0428": "Sh",
	"\u0429": "Sh",
	"\u042a": "U",
	"\u042b": "Y",
	"\u042c": "",
	"\u042d": "E",
	"\u042e": "Yu",
	"\u042f": "Ya",
	"\u0430": "a",
	"\u0431": "b",
	"\u0432": "v",
	"\u0433": "g",
	"\u0434": "d",
	"\u0435": "e",
	"\u0436": "zh",
	"\u0437": "z",
	"\u0438": "i",
	"\u0439": "j",
	"\u043a": "k",
	"\u043b": "l",
	"\u043c": "m",
	"\u043d": "n",
	"\u043e": "o",
	"\u043f": "p",
	"\u0440": "r",
	"\u0441": "s",
	"\u0442": "t",
	"\u0443": "u",
	"\u0444": "f",
	"\u0445": "h",
	"\u0446": "c",
	"\u0447": "ch",
	"\u0448": "sh",
	"\u0449": "sh",
	"\u044a": "u",
	"\u044b": "y",
	"\u044c": "",
	"\u044d": "e",
	"\u044e": "yu",
	"\u044f": "ya",
	"\u0451": "yo",
	"\u0452": "dj",
	"\u0454": "ye",
	"\u0456": "i",
	"\u0457": "yi",
	"\u0458": "j",
	"\u0459": "lj",
	"\u045a": "nj",
	"\u045b": "c",
	"\u045d": "u",
	"\u045f": "dz",
	"\u0490": "G",
	"\u0491": "g",
	"\u0492": "GH",
	"\u0493": "gh",
	"\u049a": "KH",
	"\u049b": "kh",
	"\u04a2": "NG",
	"\u04a3": "ng",
	"\u04ae": "UE",
	"\u04af": "ue",
	"\u04b0": "U",
	"\u04b1": "u",
	"\u04ba": "H",
	"\u04bb": "h",
	"\u04d8": "AE",
	"\u04d9": "ae",
	"\u04e8": "OE",
	"\u04e9": "oe",
	"\u0531": "A",
	"\u0532": "B",
	"\u0533": "G",
	"\u0534": "D",
	"\u0535": "E",
	"\u0536": "Z",
	"\u0537": "E'",
	"\u0538": "Y'",
	"\u0539": "T'",
	"\u053a": "JH",
	"\u053b": "I",
	"\u053c": "L",
	"\u053d": "X",
	"\u053e": "C'",
	"\u053f": "K",
	"\u0540": "H",
	"\u0541": "D'",
	"\u0542": "GH",
	"\u0543": "TW",
	"\u0544": "M",
	"\u0545": "Y",
	"\u0546": "N",
	"\u0547": "SH",
	"\u0549": "CH",
	"\u054a": "P",
	"\u054b": "J",
	"\u054c": "R'",
	"\u054d": "S",
	"\u054e": "V",
	"\u054f": "T",
	"\u0550": "R",
	"\u0551": "C",
	"\u0553": "P'",
	"\u0554": "Q'",
	"\u0555": "O''",
	"\u0556": "F",
	"\u0587": "EV",
	"\u0621": "a",
	"\u0622": "aa",
	"\u0623": "a",
	"\u0624": "u",
	"\u0625": "i",
	"\u0626": "e",
	"\u0627": "a",
	"\u0628": "b",
	"\u0629": "h",
	"\u062a": "t",
	"\u062b": "th",
	"\u062c": "j",
	"\u062d": "h",
	"\u062e": "kh",
	"\u062f": "d",
	"\u0630": "th",
	"\u0631": "r",
	"\u0632": "z",
	"\u0633": "s",
	"\u0634": "sh",
	"\u0635": "s",
	"\u0636": "dh",
	"\u0637": "t",
	"\u0638": "z",
	"\u0639": "a",
	"\u063a": "gh",
	"\u0641": "f",
	"\u0642": "q",
	"\u0643": "k",
	"\u0644": "l",
	"\u0645": "m",
	"\u0646": "n",
	"\u0647": "h",
	"\u0648": "w",
	"\u0649": "a",
	"\u064a": "y",
	"\u064b": "an",
	"\u064c": "on",
	"\u064d": "en",
	"\u064e": "a",
	"\u064f": "u",
	"\u0650": "e",
	"\u0652": "",
	"\u0660": "0",
	"\u0661": "1",
	"\u0662": "2",
	"\u0663": "3",
	"\u0664": "4",
	"\u0665": "5",
	"\u0666": "6",
	"\u0667": "7",
	"\u0668": "8",
	"\u0669": "9",
	"\u067e": "p",
	"\u0686": "ch",
	"\u0698": "zh",
	"\u06a9": "k",
	"\u06af": "g",
	"\u06cc": "y",
	"\u06f0": "0",
	"\u06f1": "1",
	"\u06f2": "2",
	"\u06f3": "3",
	"\u06f4": "4",
	"\u06f5": "5",
	"\u06f6": "6",
	"\u06f7": "7",
	"\u06f8": "8",
	"\u06f9": "9",
	"\u0e3f": "baht",
	"\u10d0": "a",
	"\u10d1": "b",
	"\u10d2": "g",
	"\u10d3": "d",
	"\u10d4": "e",
	"\u10d5": "v",
	"\u10d6": "z",
	"\u10d7": "t",
	"\u10d8": "i",
	"\u10d9": "k",
	"\u10da": "l",
	"\u10db": "m",
	"\u10dc": "n",
	"\u10dd": "o",
	"\u10de": "p",
	"\u10df": "zh",
	"\u10e0": "r",
	"\u10e1": "s",
	"\u10e2": "t",
	"\u10e3": "u",
	"\u10e4": "f",
	"\u10e5": "k",
	"\u10e6": "gh",
	"\u10e7": "q",
	"\u10e8": "sh",
	"\u10e9": "ch",
	"\u10ea": "ts",
	"\u10eb": "dz",
	"\u10ec": "ts",
	"\u10ed": "ch",
	"\u10ee": "kh",
	"\u10ef": "j",
	"\u10f0": "h",
	"\u1e62": "S",
	"\u1e63": "s",
	"\u1e80": "W",
	"\u1e81": "w",
	"\u1e82": "W",
	"\u1e83": "w",
	"\u1e84": "W",
	"\u1e85": "w",
	"\u1e9e": "SS",
	"\u1ea0": "A",
	"\u1ea1": "a",
	"\u1ea2": "A",
	"\u1ea3": "a",
	"\u1ea4": "A",
	"\u1ea5": "a",
	"\u1ea6": "A",
	"\u1ea7": "a",
	"\u1ea8": "A",
	"\u1ea9": "a",
	"\u1eaa": "A",
	"\u1eab": "a",
	"\u1eac": "A",
	"\u1ead": "a",
	"\u1eae": "A",
	"\u1eaf": "a",
	"\u1eb0": "A",
	"\u1eb1": "a",
	"\u1eb2": "A",
	"\u1eb3": "a",
	"\u1eb4": "A",
	"\u1eb5": "a",
	"\u1eb6": "A",
	"\u1eb7": "a",
	"\u1eb8": "E",
	"\u1eb9": "e",
	"\u1eba": "E",
	"\u1ebb": "e",
	"\u1ebc": "E",
	"\u1ebd": "e",
	"\u1ebe": "E",
	"\u1ebf": "e",
	"\u1ec0": "E",
	"\u1ec1": "e",
	"\u1ec2": "E",
	"\u1ec3": "e",
	"\u1ec4": "E",
	"\u1ec5": "e",
	"\u1ec6": "E",
	"\u1ec7": "e",
	"\u1ec8": "I",
	"\u1ec9": "i",
	"\u1eca": "I",
	"\u1ecb": "i",
	"\u1ecc": "O",
	"\u1ecd": "o",
	"\u1ece": "O",
	"\u1ecf": "o",
	"\u1ed0": "O",
	"\u1ed1": "o",
	"\u1ed2": "O",
	"\u1ed3": "o",
	"\u1ed4": "O",
	"\u1ed5": "o",
	"\u1ed6": "O",
	"\u1ed7": "o",
	"\u1ed8": "O",
	"\u1ed9": "o",
	"\u1eda": "O",
	"\u1edb": "o",
	"\u1edc": "O",
	"\u1edd": "o",
	"\u1ede": "O",
	"\u1edf": "o",
	"\u1ee0": "O",
	"\u1ee1": "o",
	"\u1ee2": "O",
	"\u1ee3": "o",
	"\u1ee4": "U",
	"\u1ee5": "u",
	"\u1ee6": "U",
	"\u1ee7": "u",
	"\u1ee8": "U",
	"\u1ee9": "u",
	"\u1eea": "U",
	"\u1eeb": "u",
	"\u1eec": "U",
	"\u1eed": "u",
	"\u1eee": "U",
	"\u1eef": "u",
	"\u1ef0": "U",
	"\u1ef1": "u",
	"\u1ef2": "Y",
	"\u1ef3": "y",
	"\u1ef4": "Y",
	"\u1ef5": "y",
	"\u1ef6": "Y",
	"\u1ef7": "y",
	"\u1ef8": "Y",
	"\u1ef9": "y",
	"\u2013": "-",
	"\u2018": "'",
	"\u2019": "'",
	"\u201c": "\\\"",
	"\u201d": "\\\"",
	"\u201e": "\\\"",
	"\u2020": "+",
	"\u2022": "*",
	"\u2026": "...",
	"\u20a0": "ecu",
	"\u20a2": "cruzeiro",
	"\u20a3": "french franc",
	"\u20a4": "lira",
	"\u20a5": "mill",
	"\u20a6": "naira",
	"\u20a7": "peseta",
	"\u20a8": "rupee",
	"\u20a9": "won",
	"\u20aa": "new shequel",
	"\u20ab": "dong",
	"\u20ac": "euro",
	"\u20ad": "kip",
	"\u20ae": "tugrik",
	"\u20af": "drachma",
	"\u20b0": "penny",
	"\u20b1": "peso",
	"\u20b2": "guarani",
	"\u20b3": "austral",
	"\u20b4": "hryvnia",
	"\u20b5": "cedi",
	"\u20b8": "kazakhstani tenge",
	"\u20b9": "indian rupee",
	"\u20ba": "turkish lira",
	"\u20bd": "russian ruble",
	"\u20bf": "bitcoin",
	"\u2120": "sm",
	"\u2122": "tm",
	"\u2202": "d",
	"\u2206": "delta",
	"\u2211": "sum",
	"\u221e": "infinity",
	"\u2665": "love",
	"\u5143": "yuan",
	"\u5186": "yen",
	"\ufdfc": "rial",
	"\ufef5": "laa",
	"\ufef7": "laa",
	"\ufef9": "lai",
	"\ufefb": "la",

}

// ---------------------------------------------------------------------------
// Locale maps
// TS source: line 16 (locales) — from config/locales.json
// Note: the "locale" key in the JSON is a human-readable name, not a mapping.
// ---------------------------------------------------------------------------

var localeMap = map[string]map[string]string{
	"bg": {
		"Й": "Y",
		"Ц": "Ts",
		"Щ": "Sht",
		"Ъ": "A",
		"Ь": "Y",
		"й": "y",
		"ц": "ts",
		"щ": "sht",
		"ъ": "a",
		"ь": "y",
	},
	"de": {
		"Ä": "AE",
		"ä": "ae",
		"Ö": "OE",
		"ö": "oe",
		"Ü": "UE",
		"ü": "ue",
		"ß": "ss",
		"%": "prozent",
		"&": "und",
		"|": "oder",
		"∑": "summe",
		"∞": "unendlich",
		"♥": "liebe",
	},
	"es": {
		"%": "por ciento",
		"&": "y",
		"<": "menor que",
		">": "mayor que",
		"|": "o",
		"¢": "centavos",
		"£": "libras",
		"¤": "moneda",
		"₣": "francos",
		"∑": "suma",
		"∞": "infinito",
		"♥": "amor",
	},
	"fr": {
		"%": "pourcent",
		"&": "et",
		"<": "plus petit",
		">": "plus grand",
		"|": "ou",
		"¢": "centime",
		"£": "livre",
		"¤": "devise",
		"₣": "franc",
		"∑": "somme",
		"∞": "infini",
		"♥": "amour",
	},
	"pt": {
		"%": "porcento",
		"&": "e",
		"<": "menor",
		">": "maior",
		"|": "ou",
		"¢": "centavo",
		"∑": "soma",
		"£": "libra",
		"∞": "infinito",
		"♥": "amor",
	},
	"uk": {
		"И": "Y",
		"и": "y",
		"Й": "Y",
		"й": "y",
		"Ц": "Ts",
		"ц": "ts",
		"Х": "Kh",
		"х": "kh",
		"Щ": "Shch",
		"щ": "shch",
		"Г": "H",
		"г": "h",
	},
	"vi": {
		"Đ": "D",
		"đ": "d",
	},
	"da": {
		"Ø": "OE",
		"ø": "oe",
		"Å": "AA",
		"å": "aa",
		"%": "procent",
		"&": "og",
		"|": "eller",
		"$": "dollar",
		"<": "mindre end",
		">": "større end",
	},
	"nb": {
		"&": "og",
		"Å": "AA",
		"Æ": "AE",
		"Ø": "OE",
		"å": "aa",
		"æ": "ae",
		"ø": "oe",
	},
	"it": {
		"&": "e",
	},
	"nl": {
		"&": "en",
	},
	"sv": {
		"&": "och",
		"Å": "AA",
		"Ä": "AE",
		"Ö": "OE",
		"å": "aa",
		"ä": "ae",
		"ö": "oe",
	},
}
