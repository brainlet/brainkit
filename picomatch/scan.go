package picomatch

// Ported from: picomatch/lib/scan.js

// ScanToken represents a single token from scanning a glob pattern.
// JS source: scan.js line 75
type ScanToken struct {
	Value      string
	Depth      int
	IsGlob     bool
	IsPrefix   bool
	IsGlobstar bool
	IsBrace    bool
	IsExtglob  bool
	IsBracket  bool
	Negated    bool
	Backslashes bool
}

// ScanState is the result of scanning a glob pattern.
// JS source: scan.js lines 327-340
type ScanState struct {
	Prefix        string
	Input         string
	Start         int
	Base          string
	Glob          string
	IsBrace       bool
	IsBracket     bool
	IsGlob        bool
	IsExtglob     bool
	IsGlobstar    bool
	Negated       bool
	NegatedExtglob bool
	// Optional fields (populated when opts.Tokens or opts.Parts is true)
	MaxDepth int
	Tokens   []ScanToken
	Slashes  []int
	Parts    []string
}

// ScanOptions configures the scan behavior.
type ScanOptions struct {
	Parts     bool
	ScanToEnd bool
	Tokens    bool
	Noext     bool
	Nonegate  bool
	Noparen   bool
	Unescape  bool
}

// isPathSeparator returns true if code is / or \.
// JS source: scan.js lines 22-24
func isPathSeparator(code byte) bool {
	return code == CHAR_FORWARD_SLASH || code == CHAR_BACKWARD_SLASH
}

// scanTokenDepth sets the depth on a scan token.
// JS source: scan.js lines 26-30
func scanTokenDepth(token *ScanToken) {
	if !token.IsPrefix {
		if token.IsGlobstar {
			token.Depth = 1<<31 - 1 // MaxInt for Infinity
		} else {
			token.Depth = 1
		}
	}
}

// Scan quickly scans a glob pattern and returns an object with useful properties.
// JS source: scan.js lines 49-389
func Scan(input string, scanOpts *ScanOptions) ScanState {
	opts := &ScanOptions{}
	if scanOpts != nil {
		*opts = *scanOpts
	}

	length := len(input) - 1
	scanToEnd := opts.Parts || opts.ScanToEnd

	var slashes []int
	var tokens []ScanToken
	var parts []string

	str := input
	index := -1
	start := 0
	lastIndex := 0
	isBrace := false
	isBracket := false
	isGlob := false
	isExtglob := false
	isGlobstar := false
	braceEscaped := false
	backslashes := false
	negated := false
	negatedExtglob := false
	finished := false
	braces := 0
	var prev byte
	var code byte
	token := ScanToken{Value: "", Depth: 0, IsGlob: false}

	eos := func() bool { return index >= length }
	peek := func() byte {
		if index+1 < len(str) {
			return str[index+1]
		}
		return 0
	}
	advance := func() byte {
		prev = code
		index++
		if index < len(str) {
			return str[index]
		}
		return 0
	}

	for index < length {
		code = advance()
		var next byte
		_ = next

		// JS source: scan.js lines 88-96 — Backslash handling
		if code == CHAR_BACKWARD_SLASH {
			backslashes = true
			token.Backslashes = true
			code = advance()

			if code == CHAR_LEFT_CURLY_BRACE {
				braceEscaped = true
			}
			continue
		}

		// JS source: scan.js lines 98-154 — Brace handling
		if braceEscaped || code == CHAR_LEFT_CURLY_BRACE {
			braces++

			for !eos() {
				code = advance()
				if code == 0 {
					break
				}

				if code == CHAR_BACKWARD_SLASH {
					backslashes = true
					token.Backslashes = true
					advance()
					continue
				}

				if code == CHAR_LEFT_CURLY_BRACE {
					braces++
					continue
				}

				if !braceEscaped && code == CHAR_DOT {
					code = advance()
					if code == CHAR_DOT {
						isBrace = true
						token.IsBrace = true
						isGlob = true
						token.IsGlob = true
						finished = true

						if scanToEnd {
							continue
						}
						break
					}
				}

				if !braceEscaped && code == CHAR_COMMA {
					isBrace = true
					token.IsBrace = true
					isGlob = true
					token.IsGlob = true
					finished = true

					if scanToEnd {
						continue
					}
					break
				}

				if code == CHAR_RIGHT_CURLY_BRACE {
					braces--

					if braces == 0 {
						braceEscaped = false
						isBrace = true
						token.IsBrace = true
						finished = true
						break
					}
				}
			}

			if scanToEnd {
				continue
			}
			break
		}

		// JS source: scan.js lines 156-169 — Forward slash
		if code == CHAR_FORWARD_SLASH {
			slashes = append(slashes, index)
			tokens = append(tokens, token)
			token = ScanToken{Value: "", Depth: 0, IsGlob: false}

			if finished {
				continue
			}
			if prev == CHAR_DOT && index == start+1 {
				start += 2
				continue
			}

			lastIndex = index + 1
			continue
		}

		// JS source: scan.js lines 171-203 — Extglob detection
		if !opts.Noext {
			isExtglobChar := code == CHAR_PLUS ||
				code == CHAR_AT ||
				code == CHAR_ASTERISK ||
				code == CHAR_QUESTION_MARK ||
				code == CHAR_EXCLAMATION_MARK

			if isExtglobChar && peek() == CHAR_LEFT_PARENTHESES {
				isGlob = true
				token.IsGlob = true
				isExtglob = true
				token.IsExtglob = true
				finished = true
				if code == CHAR_EXCLAMATION_MARK && index == start {
					negatedExtglob = true
				}

				if scanToEnd {
					for !eos() {
						code = advance()
						if code == 0 {
							break
						}

						if code == CHAR_BACKWARD_SLASH {
							backslashes = true
							token.Backslashes = true
							advance()
							continue
						}

						if code == CHAR_RIGHT_PARENTHESES {
							isGlob = true
							token.IsGlob = true
							finished = true
							break
						}
					}
					continue
				}
				break
			}
		}

		// JS source: scan.js lines 206-215 — Asterisk
		if code == CHAR_ASTERISK {
			if prev == CHAR_ASTERISK {
				isGlobstar = true
				token.IsGlobstar = true
			}
			isGlob = true
			token.IsGlob = true
			finished = true

			if scanToEnd {
				continue
			}
			break
		}

		// JS source: scan.js lines 217-225 — Question mark
		if code == CHAR_QUESTION_MARK {
			isGlob = true
			token.IsGlob = true
			finished = true

			if scanToEnd {
				continue
			}
			break
		}

		// JS source: scan.js lines 227-248 — Left square bracket
		if code == CHAR_LEFT_SQUARE_BRACKET {
			for !eos() {
				next = advance()
				if next == 0 {
					break
				}

				if next == CHAR_BACKWARD_SLASH {
					backslashes = true
					token.Backslashes = true
					advance()
					continue
				}

				if next == CHAR_RIGHT_SQUARE_BRACKET {
					isBracket = true
					token.IsBracket = true
					isGlob = true
					token.IsGlob = true
					finished = true
					break
				}
			}

			if scanToEnd {
				continue
			}
			break
		}

		// JS source: scan.js lines 250-254 — Negation
		if !opts.Nonegate && code == CHAR_EXCLAMATION_MARK && index == start {
			negated = true
			token.Negated = true
			start++
			continue
		}

		// JS source: scan.js lines 256-275 — Left parenthesis
		if !opts.Noparen && code == CHAR_LEFT_PARENTHESES {
			isGlob = true
			token.IsGlob = true

			if scanToEnd {
				for !eos() {
					code = advance()
					if code == 0 {
						break
					}

					if code == CHAR_LEFT_PARENTHESES {
						backslashes = true
						token.Backslashes = true
						advance()
						continue
					}

					if code == CHAR_RIGHT_PARENTHESES {
						finished = true
						break
					}
				}
				continue
			}
			break
		}

		// JS source: scan.js lines 277-285 — Already a glob
		if isGlob {
			finished = true

			if scanToEnd {
				continue
			}
			break
		}
	}

	// JS source: scan.js lines 288-291 — noext disables extglob/glob
	if opts.Noext {
		isExtglob = false
		isGlob = false
	}

	// JS source: scan.js lines 293-317 — Compute base, prefix, glob
	base := str
	prefix := ""
	glob := ""

	if start > 0 {
		prefix = str[:start]
		str = str[start:]
		lastIndex -= start
	}

	if isGlob && lastIndex > 0 {
		base = str[:lastIndex]
		glob = str[lastIndex:]
	} else if isGlob {
		base = ""
		glob = str
	} else {
		base = str
	}

	if base != "" && base != "/" && base != input {
		if len(base) > 0 && isPathSeparator(base[len(base)-1]) {
			base = base[:len(base)-1]
		}
	}

	// JS source: scan.js lines 319-325 — Unescape
	if opts.Unescape {
		if glob != "" {
			glob = removeBackslashesManual(glob)
		}
		if backslashes {
			base = removeBackslashesManual(base)
		}
	}

	// JS source: scan.js lines 327-340 — Build state
	state := ScanState{
		Prefix:         prefix,
		Input:          input,
		Start:          start,
		Base:           base,
		Glob:           glob,
		IsBrace:        isBrace,
		IsBracket:      isBracket,
		IsGlob:         isGlob,
		IsExtglob:      isExtglob,
		IsGlobstar:     isGlobstar,
		Negated:        negated,
		NegatedExtglob: negatedExtglob,
	}

	// JS source: scan.js lines 342-348 — Tokens
	if opts.Tokens {
		state.MaxDepth = 0
		if index < len(str) && !isPathSeparator(code) {
			tokens = append(tokens, token)
		}
		state.Tokens = tokens
	}

	// JS source: scan.js lines 350-386 — Parts
	if opts.Parts || opts.Tokens {
		var prevIndex int
		hasPrevIndex := false

		for idx := 0; idx < len(slashes); idx++ {
			var n int
			if hasPrevIndex {
				n = prevIndex + 1
			} else {
				n = start
			}
			i := slashes[idx]
			value := input[n:i]

			if opts.Tokens {
				if idx == 0 && start != 0 {
					tokens[idx].IsPrefix = true
					tokens[idx].Value = prefix
				} else {
					tokens[idx].Value = value
				}
				scanTokenDepth(&tokens[idx])
				state.MaxDepth += tokens[idx].Depth
			}

			if idx != 0 || value != "" {
				parts = append(parts, value)
			}

			prevIndex = i
			hasPrevIndex = true
		}

		if hasPrevIndex && prevIndex+1 < len(input) {
			value := input[prevIndex+1:]
			parts = append(parts, value)

			if opts.Tokens && len(tokens) > 0 {
				tokens[len(tokens)-1].Value = value
				scanTokenDepth(&tokens[len(tokens)-1])
				state.MaxDepth += tokens[len(tokens)-1].Depth
			}
		}

		state.Slashes = slashes
		state.Parts = parts
	}

	return state
}
