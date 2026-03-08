package picomatch

// Ported from: picomatch/lib/parse.js (1085 lines)

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
)

// Token represents a parsed token in the glob pattern.
// JS source: parse.js — used throughout the tokenizer
type Token struct {
	Type        string
	Value       string
	Output      string
	OutputIndex int
	TokensIndex int
	Prev        *Token
	Suffix      string

	// Extglob fields
	Extglob    bool
	Inner      string
	Conditions int
	Parens     int
	Open       string
	Close      string

	// Bracket fields
	Posix bool

	// Brace fields
	Dots  bool
	Comma bool

	// Star fields
	Star bool
}

// ParseState is the result of parsing a glob pattern.
// JS source: parse.js lines 111-127
type ParseState struct {
	Input          string
	Index          int
	Start          int
	Dot            bool
	Consumed       string
	Output         string
	Prefix         string
	Backtrack      bool
	Negated        bool
	NegatedExtglob bool
	Brackets       int
	Braces         int
	Parens         int
	Quotes         int
	Globstar       bool
	Tokens         []*Token
	Fastpaths      bool
}

// --- Compiled regex for parse ---
var reNonSpecialChars = regexp2.MustCompile(regexNonSpecialCharsPattern, regexp2.None)
var reSpecialCharsBackref = regexp2.MustCompile(regexSpecialCharsBackrefPattern, regexp2.None)

// expandRange creates a character class range from args.
// JS source: parse.js lines 22-38
func expandRange(args []string, opts *Options) string {
	if opts != nil && opts.ExpandRange != nil {
		return opts.ExpandRange(args, opts)
	}

	sort.Strings(args)
	value := "[" + strings.Join(args, "-") + "]"

	// Validate the regex
	_, err := regexp2.Compile(value, regexp2.None)
	if err != nil {
		// Invalid range — escape and join with ..
		escaped := make([]string, len(args))
		for i, v := range args {
			escaped[i] = escapeRegex(v)
		}
		return strings.Join(escaped, "..")
	}

	return value
}

// syntaxError creates a syntax error message.
// JS source: parse.js lines 44-46
func syntaxError(typ, char string) string {
	return fmt.Sprintf(`Missing %s: "%s" - use "\\\\%s" to match literal characters`, typ, char, char)
}

// Parse parses a glob pattern into a ParseState with regex output.
// JS source: parse.js lines 55-991
func Parse(input string, opts *Options) *ParseState {
	if opts == nil {
		opts = &Options{}
	}

	if r, ok := REPLACEMENTS[input]; ok {
		input = r
	}

	maxLen := MaxLength
	if opts.MaxLength > 0 && opts.MaxLength < maxLen {
		maxLen = opts.MaxLength
	}

	inputLen := len(input)
	if inputLen > maxLen {
		panic(fmt.Sprintf("Input length: %d, exceeds maximum allowed length: %d", inputLen, maxLen))
	}

	// JS source: parse.js lines 70-127
	bos := &Token{Type: "bos", Value: "", Output: opts.Prepend}
	tokens := []*Token{bos}

	capture := "?:"
	if opts.Capture {
		capture = ""
	}

	// Create constants based on platform
	// JS source: parse.js lines 76-92
	PLATFORM_CHARS := GlobChars(opts.Windows)
	EXTGLOB_CHARS := ExtglobChars(PLATFORM_CHARS)

	pc := PLATFORM_CHARS // shorthand
	dotLiteral := pc.DOT_LITERAL
	plusLiteral := pc.PLUS_LITERAL
	slashLiteral := pc.SLASH_LITERAL
	oneChar := pc.ONE_CHAR
	dotsSl := pc.DOTS_SLASH
	noDot := pc.NO_DOT
	noDotSlash := pc.NO_DOT_SLASH
	noDotsSlash := pc.NO_DOTS_SLASH
	qmark := pc.QMARK
	qmarkNoDot := pc.QMARK_NO_DOT
	star := pc.STAR
	startAnchor := pc.START_ANCHOR

	_ = dotsSl // used in globstar

	// JS source: parse.js lines 94-96
	globstarFn := func(dotOpt bool) string {
		ds := dotLiteral
		if dotOpt {
			ds = dotsSl
		}
		return "(" + capture + "(?:(?!" + startAnchor + ds + ").)*?)"
	}

	// JS source: parse.js lines 98-104
	nodot := noDot
	if opts.Dot {
		nodot = ""
	}
	qmarkND := qmarkNoDot
	if opts.Dot {
		qmarkND = qmark
	}
	starStr := star
	if opts.Bash {
		starStr = globstarFn(opts.Dot)
	}
	if opts.Capture {
		starStr = "(" + starStr + ")"
	}

	// minimatch compat
	// JS source: parse.js lines 107-109
	if opts.Noext {
		opts.Noextglob = true
	}

	// JS source: parse.js lines 111-127
	state := &ParseState{
		Input:    input,
		Index:    -1,
		Start:    0,
		Dot:      opts.Dot,
		Consumed: "",
		Output:   "",
		Prefix:   "",
		Backtrack: false,
		Negated:  false,
		Brackets: 0,
		Braces:   0,
		Parens:   0,
		Quotes:   0,
		Globstar: false,
		Tokens:   tokens,
	}

	// JS source: parse.js line 129
	input = removePrefix(input, state)
	inputLen = len(input)

	var extglobs []*Token
	var braces []*Token
	var stack []string
	prev := bos
	var value string

	// --- Tokenizing helpers ---
	// JS source: parse.js lines 142-153

	eos := func() bool { return state.Index == inputLen-1 }
	peek := func(n int) string {
		if state.Index+n < inputLen {
			return string(input[state.Index+n])
		}
		return ""
	}
	advance := func() string {
		state.Index++
		if state.Index < inputLen {
			return string(input[state.Index])
		}
		return ""
	}
	remaining := func() string {
		if state.Index+1 < inputLen {
			return input[state.Index+1:]
		}
		return ""
	}
	consume := func(val string, num int) {
		state.Consumed += val
		state.Index += num
	}

	appendTok := func(tok *Token) {
		out := tok.Output
		if out == "" {
			out = tok.Value
		}
		state.Output += out
		consume(tok.Value, 0)
	}

	// negate — JS source: parse.js lines 156-172
	negate := func() bool {
		count := 1
		for peek(1) == "!" && (peek(2) != "(" || peek(3) == "?") {
			advance()
			state.Start++
			count++
		}
		if count%2 == 0 {
			return false
		}
		state.Negated = true
		state.Start++
		return true
	}

	increment := func(typ string) {
		switch typ {
		case "parens":
			state.Parens++
		case "braces":
			state.Braces++
		case "brackets":
			state.Brackets++
		}
		stack = append(stack, typ)
	}

	decrement := func(typ string) {
		switch typ {
		case "parens":
			state.Parens--
		case "braces":
			state.Braces--
		case "brackets":
			state.Brackets--
		}
		if len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}

	// push — JS source: parse.js lines 192-220
	var push func(tok *Token)
	push = func(tok *Token) {
		// JS source: parse.js lines 193-204
		if prev.Type == "globstar" {
			isBrace := state.Braces > 0 && (tok.Type == "comma" || tok.Type == "brace")
			isExtg := tok.Extglob || (len(extglobs) > 0 && (tok.Type == "pipe" || tok.Type == "paren"))

			if tok.Type != "slash" && tok.Type != "paren" && !isBrace && !isExtg {
				state.Output = state.Output[:len(state.Output)-len(prev.Output)]
				prev.Type = "star"
				prev.Value = "*"
				prev.Output = starStr
				state.Output += prev.Output
			}
		}

		// JS source: parse.js lines 206-208
		if len(extglobs) > 0 && tok.Type != "paren" {
			extglobs[len(extglobs)-1].Inner += tok.Value
		}

		// JS source: parse.js lines 210-219
		if tok.Value != "" || tok.Output != "" {
			appendTok(tok)
		}
		if prev != nil && prev.Type == "text" && tok.Type == "text" {
			if prev.Output != "" {
				prev.Output += tok.Value
			} else {
				prev.Output = prev.Value + tok.Value
			}
			prev.Value += tok.Value
			return
		}

		tok.Prev = prev
		tokens = append(tokens, tok)
		prev = tok
	}

	// extglobOpen — JS source: parse.js lines 222-234
	extglobOpen := func(typ string, val string) {
		ec := EXTGLOB_CHARS[val[0]]
		token := &Token{
			Type:       ec.Type,
			Conditions: 1,
			Inner:      "",
			Open:       ec.Open,
			Close:      ec.Close,
			Prev:       prev,
			Parens:     state.Parens,
			Output:     state.Output,
		}

		output := ""
		if opts.Capture {
			output = "("
		}
		output += token.Open

		increment("parens")
		push(&Token{Type: typ, Value: val, Output: func() string {
			if state.Output != "" {
				return ""
			}
			return oneChar
		}()})
		push(&Token{Type: "paren", Extglob: true, Value: advance(), Output: output})
		extglobs = append(extglobs, token)
	}

	// extglobClose — JS source: parse.js lines 236-269
	extglobClose := func(token *Token) {
		output := token.Close
		if opts.Capture {
			output += ")"
		}

		if token.Type == "negate" {
			extglobStar := starStr

			if token.Inner != "" && len(token.Inner) > 1 && strings.Contains(token.Inner, "/") {
				extglobStar = globstarFn(opts.Dot)
			}

			if extglobStar != starStr || eos() || matchStr(`^\)+$`, remaining()) {
				output = token.Close + ")$))" + extglobStar
			}

			if strings.Contains(token.Inner, "*") {
				rest := remaining()
				if rest != "" && matchStr(`^\.[^\\/\.]+$`, rest) {
					expr := Parse(rest, &Options{
						Fastpaths: boolPtr(false),
						Windows:   opts.Windows,
						Dot:       opts.Dot,
					})
					output = token.Close + ")" + expr.Output + ")" + extglobStar + ")"
				}
			}

			if token.Prev != nil && token.Prev.Type == "bos" {
				state.NegatedExtglob = true
			}
		}

		push(&Token{Type: "paren", Extglob: true, Value: value, Output: output})
		decrement("parens")
	}

	// =====================================================
	// Fast paths
	// JS source: parse.js lines 275-324
	// =====================================================

	if opts.Fastpaths == nil || *opts.Fastpaths {
		if !matchStr(`(^[*!]|[/()[\]{}")])`, input) {
			backslashesLocal := false

			// Use regex replacement for fastpath
			// JS source: parse.js lines 278-305
			output := fastpathReplace(input, func(m string, esc string, chars string, first string, rest string, idx int) string {
				if first == `\` {
					backslashesLocal = true
					return m
				}

				if first == "?" {
					if esc != "" {
						return esc + first + strings.Repeat(qmark, len(rest))
					}
					if idx == 0 {
						return qmarkND + strings.Repeat(qmark, len(rest))
					}
					return strings.Repeat(qmark, len(chars))
				}

				if first == "." {
					return strings.Repeat(dotLiteral, len(chars))
				}

				if first == "*" {
					if esc != "" {
						s := starStr
						if rest == "" {
							s = ""
						}
						return esc + first + s
					}
					return starStr
				}
				if esc != "" {
					return m
				}
				return `\` + m
			})

			if backslashesLocal {
				if opts.Unescape {
					output = strings.ReplaceAll(output, `\`, "")
				} else {
					output = replaceBackslashRuns(output)
				}
			}

			if output == input && opts.Contains {
				state.Output = input
				return state
			}

			state.Output = wrapOutput(output, state, opts)
			return state
		}
	}

	// =====================================================
	// Tokenize input until we reach end-of-string
	// JS source: parse.js lines 330-953
	// =====================================================

	for !eos() {
		value = advance()

		// Null byte
		if value == "\x00" {
			continue
		}

		// --- Escaped characters ---
		// JS source: parse.js lines 341-380
		if value == `\` {
			next := peek(1)

			if next == "/" && !opts.Bash {
				continue
			}

			if next == "." || next == ";" {
				continue
			}

			if next == "" {
				value += `\`
				push(&Token{Type: "text", Value: value})
				continue
			}

			// Collapse slashes
			// JS source: parse.js lines 359-368
			rem := remaining()
			slashMatch := matchPrefix(`^\\+`, rem)
			slashes := 0

			if slashMatch != "" && len(slashMatch) > 2 {
				slashes = len(slashMatch)
				state.Index += slashes
				if slashes%2 != 0 {
					value += `\`
				}
			}

			if opts.Unescape {
				value = advance()
			} else {
				value += advance()
			}

			if state.Brackets == 0 {
				push(&Token{Type: "text", Value: value})
				continue
			}
		}

		// --- Inside bracket expression ---
		// JS source: parse.js lines 387-427
		if state.Brackets > 0 && (value != "]" || prev.Value == "[" || prev.Value == "[^") {
			if !opts.NoPosix && value == ":" {
				inner := prev.Value[1:]
				if strings.Contains(inner, "[") {
					prev.Posix = true

					if strings.Contains(inner, ":") {
						idx := strings.LastIndex(prev.Value, "[")
						pre := prev.Value[:idx]
						rest := prev.Value[idx+2:]
						if posix, ok := POSIX_REGEX_SOURCE[rest]; ok {
							prev.Value = pre + posix
							state.Backtrack = true
							advance()

							if bos.Output == "" && len(tokens) > 1 && tokens[1] == prev {
								bos.Output = oneChar
							}
							continue
						}
					}
				}
			}

			if (value == "[" && peek(1) != ":") || (value == "-" && peek(1) == "]") {
				value = `\` + value
			}

			if value == "]" && (prev.Value == "[" || prev.Value == "[^") {
				value = `\` + value
			}

			if opts.Posix && value == "!" && prev.Value == "[" {
				value = "^"
			}

			prev.Value += value
			appendTok(&Token{Value: value})
			continue
		}

		// --- Inside quoted string ---
		// JS source: parse.js lines 434-439
		if state.Quotes == 1 && value != `"` {
			value = escapeRegex(value)
			prev.Value += value
			appendTok(&Token{Value: value})
			continue
		}

		// --- Double quotes ---
		// JS source: parse.js lines 445-451
		if value == `"` {
			if state.Quotes == 1 {
				state.Quotes = 0
			} else {
				state.Quotes = 1
			}
			if opts.KeepQuotes {
				push(&Token{Type: "text", Value: value})
			}
			continue
		}

		// --- Parentheses ---
		// JS source: parse.js lines 457-477
		if value == "(" {
			increment("parens")
			push(&Token{Type: "paren", Value: value})
			continue
		}

		if value == ")" {
			if state.Parens == 0 && opts.StrictBrackets {
				panic(syntaxError("opening", "("))
			}

			if len(extglobs) > 0 {
				extglob := extglobs[len(extglobs)-1]
				if state.Parens == extglob.Parens+1 {
					extglobClose(extglobs[len(extglobs)-1])
					extglobs = extglobs[:len(extglobs)-1]
					continue
				}
			}

			out := ")"
			if state.Parens > 0 {
				out = ")"
			} else {
				out = `\)`
			}
			push(&Token{Type: "paren", Value: value, Output: out})
			decrement("parens")
			continue
		}

		// --- Square brackets ---
		// JS source: parse.js lines 483-543
		if value == "[" {
			if opts.Nobracket || !strings.Contains(remaining(), "]") {
				if !opts.Nobracket && opts.StrictBrackets {
					panic(syntaxError("closing", "]"))
				}
				value = `\` + value
			} else {
				increment("brackets")
			}

			push(&Token{Type: "bracket", Value: value})
			continue
		}

		if value == "]" {
			if opts.Nobracket || (prev != nil && prev.Type == "bracket" && len(prev.Value) == 1) {
				push(&Token{Type: "text", Value: value, Output: `\` + value})
				continue
			}

			if state.Brackets == 0 {
				if opts.StrictBrackets {
					panic(syntaxError("opening", "["))
				}
				push(&Token{Type: "text", Value: value, Output: `\` + value})
				continue
			}

			decrement("brackets")

			prevValue := prev.Value[1:]
			if !prev.Posix && len(prevValue) > 0 && prevValue[0] == '^' && !strings.Contains(prevValue, "/") {
				value = "/" + value
			}

			prev.Value += value
			appendTok(&Token{Value: value})

			// JS source: parse.js lines 525-543
			if opts.LiteralBrackets == nil || !*opts.LiteralBrackets {
				if hasRegexChars(prevValue) {
					continue
				}
			}

			escaped := escapeRegex(prev.Value)
			state.Output = state.Output[:len(state.Output)-len(prev.Value)]

			if opts.LiteralBrackets != nil && *opts.LiteralBrackets {
				state.Output += escaped
				prev.Value = escaped
				continue
			}

			prev.Value = "(" + capture + escaped + "|" + prev.Value + ")"
			state.Output += prev.Value
			continue
		}

		// --- Braces ---
		// JS source: parse.js lines 550-609
		if value == "{" && !opts.Nobrace {
			increment("braces")

			open := &Token{
				Type:        "brace",
				Value:       value,
				Output:      "(",
				OutputIndex: len(state.Output),
				TokensIndex: len(state.Tokens),
			}

			braces = append(braces, open)
			push(open)
			continue
		}

		if value == "}" {
			var brace *Token
			if len(braces) > 0 {
				brace = braces[len(braces)-1]
			}

			if opts.Nobrace || brace == nil {
				push(&Token{Type: "text", Value: value, Output: value})
				continue
			}

			output := ")"

			if brace.Dots {
				arr := make([]*Token, len(tokens))
				copy(arr, tokens)
				var rng []string

				for i := len(arr) - 1; i >= 0; i-- {
					tokens = tokens[:len(tokens)-1]
					if arr[i].Type == "brace" {
						break
					}
					if arr[i].Type != "dots" {
						rng = append([]string{arr[i].Value}, rng...)
					}
				}

				output = expandRange(rng, opts)
				state.Backtrack = true
			}

			if !brace.Comma && !brace.Dots {
				out := state.Output[:brace.OutputIndex]
				toks := state.Tokens[brace.TokensIndex:]
				brace.Value = `\{`
				brace.Output = `\{`
				value = `\}`
				output = `\}`
				state.Output = out
				for _, t := range toks {
					if t.Output != "" {
						state.Output += t.Output
					} else {
						state.Output += t.Value
					}
				}
			}

			push(&Token{Type: "brace", Value: value, Output: output})
			decrement("braces")
			if len(braces) > 0 {
				braces = braces[:len(braces)-1]
			}
			continue
		}

		// --- Pipes ---
		// JS source: parse.js lines 615-621
		if value == "|" {
			if len(extglobs) > 0 {
				extglobs[len(extglobs)-1].Conditions++
			}
			push(&Token{Type: "text", Value: value})
			continue
		}

		// --- Commas ---
		// JS source: parse.js lines 627-638
		if value == "," {
			output := value

			if len(braces) > 0 && len(stack) > 0 && stack[len(stack)-1] == "braces" {
				brace := braces[len(braces)-1]
				brace.Comma = true
				output = "|"
			}

			push(&Token{Type: "comma", Value: value, Output: output})
			continue
		}

		// --- Slashes ---
		// JS source: parse.js lines 644-660
		if value == "/" {
			if prev.Type == "dot" && state.Index == state.Start+1 {
				state.Start = state.Index + 1
				state.Consumed = ""
				state.Output = ""
				tokens = tokens[:len(tokens)-1]
				prev = bos
				continue
			}

			push(&Token{Type: "slash", Value: value, Output: slashLiteral})
			continue
		}

		// --- Dots ---
		// JS source: parse.js lines 666-684
		if value == "." {
			if state.Braces > 0 && prev.Type == "dot" {
				if prev.Value == "." {
					prev.Output = dotLiteral
				}
				if len(braces) > 0 {
					brace := braces[len(braces)-1]
					prev.Type = "dots"
					prev.Output += value
					prev.Value += value
					brace.Dots = true
				}
				continue
			}

			if (state.Braces+state.Parens) == 0 && prev.Type != "bos" && prev.Type != "slash" {
				push(&Token{Type: "text", Value: value, Output: dotLiteral})
				continue
			}

			push(&Token{Type: "dot", Value: value, Output: dotLiteral})
			continue
		}

		// --- Question marks ---
		// JS source: parse.js lines 690-716
		if value == "?" {
			isGroup := prev != nil && prev.Value == "("
			if !isGroup && !opts.Noextglob && peek(1) == "(" && peek(2) != "?" {
				extglobOpen("qmark", value)
				continue
			}

			if prev != nil && prev.Type == "paren" {
				next := peek(1)
				out := value

				if (prev.Value == "(" && !matchStr(`[!=<:]`, next)) || (next == "<" && !matchStr(`<([!=]|\w+>)`, remaining())) {
					out = `\` + value
				}

				push(&Token{Type: "text", Value: value, Output: out})
				continue
			}

			if !opts.Dot && (prev.Type == "slash" || prev.Type == "bos") {
				push(&Token{Type: "qmark", Value: value, Output: qmarkNoDot})
				continue
			}

			push(&Token{Type: "qmark", Value: value, Output: qmark})
			continue
		}

		// --- Exclamation ---
		// JS source: parse.js lines 722-734
		if value == "!" {
			if !opts.Noextglob && peek(1) == "(" {
				if peek(2) != "?" || !matchStr(`[!=<:]`, peek(3)) {
					extglobOpen("negate", value)
					continue
				}
			}

			if !opts.Nonegate && state.Index == 0 {
				negate()
				continue
			}
		}

		// --- Plus ---
		// JS source: parse.js lines 740-758
		if value == "+" {
			if !opts.Noextglob && peek(1) == "(" && peek(2) != "?" {
				extglobOpen("plus", value)
				continue
			}

			if (prev != nil && prev.Value == "(") || opts.Regex == boolPtr(false) {
				push(&Token{Type: "plus", Value: value, Output: plusLiteral})
				continue
			}

			if (prev != nil && (prev.Type == "bracket" || prev.Type == "paren" || prev.Type == "brace")) || state.Parens > 0 {
				push(&Token{Type: "plus", Value: value})
				continue
			}

			push(&Token{Type: "plus", Value: plusLiteral})
			continue
		}

		// --- At sign ---
		// JS source: parse.js lines 764-772
		if value == "@" {
			if !opts.Noextglob && peek(1) == "(" && peek(2) != "?" {
				push(&Token{Type: "at", Extglob: true, Value: value, Output: ""})
				continue
			}

			push(&Token{Type: "text", Value: value})
			continue
		}

		// --- Plain text (not *) ---
		// JS source: parse.js lines 778-791
		if value != "*" {
			if value == "$" || value == "^" {
				value = `\` + value
			}

			rem := remaining()
			nonSpecialMatch := matchPrefix(regexNonSpecialCharsPattern, rem)
			if nonSpecialMatch != "" {
				value += nonSpecialMatch
				state.Index += len(nonSpecialMatch)
			}

			push(&Token{Type: "text", Value: value})
			continue
		}

		// =====================================================
		// Stars
		// JS source: parse.js lines 797-953
		// =====================================================

		if prev != nil && (prev.Type == "globstar" || prev.Star) {
			prev.Type = "star"
			prev.Star = true
			prev.Value += value
			prev.Output = starStr
			state.Backtrack = true
			state.Globstar = true
			consume(value, 0)
			continue
		}

		rest := remaining()
		if !opts.Noextglob && matchStr(`^\([^?]`, rest) {
			extglobOpen("star", value)
			continue
		}

		if prev.Type == "star" {
			if opts.Noglobstar {
				consume(value, 0)
				continue
			}

			prior := prev.Prev
			before := prior.Prev
			isStart := prior.Type == "slash" || prior.Type == "bos"
			afterStar := before != nil && (before.Type == "star" || before.Type == "globstar")

			if opts.Bash && (!isStart || (len(rest) > 0 && rest[0] != '/')) {
				push(&Token{Type: "star", Value: value, Output: ""})
				continue
			}

			isBr := state.Braces > 0 && (prior.Type == "comma" || prior.Type == "brace")
			isExtg := len(extglobs) > 0 && (prior.Type == "pipe" || prior.Type == "paren")
			if !isStart && prior.Type != "paren" && !isBr && !isExtg {
				push(&Token{Type: "star", Value: value, Output: ""})
				continue
			}

			// Strip consecutive /**/
			for strings.HasPrefix(rest, "/**") {
				after := ""
				if state.Index+4 < inputLen {
					after = string(input[state.Index+4])
				}
				if after != "" && after != "/" {
					break
				}
				rest = rest[3:]
				consume("/**", 3)
			}

			if prior.Type == "bos" && eos() {
				prev.Type = "globstar"
				prev.Value += value
				prev.Output = globstarFn(opts.Dot)
				state.Output = prev.Output
				state.Globstar = true
				consume(value, 0)
				continue
			}

			if prior.Type == "slash" && prior.Prev.Type != "bos" && !afterStar && eos() {
				state.Output = state.Output[:len(state.Output)-len(prior.Output+prev.Output)]
				prior.Output = "(?:" + prior.Output

				strictEnd := ")"
				if opts.StrictSlashes {
					strictEnd = ")"
				} else {
					strictEnd = "|$)"
				}

				prev.Type = "globstar"
				prev.Output = globstarFn(opts.Dot) + strictEnd
				prev.Value += value
				state.Globstar = true
				state.Output += prior.Output + prev.Output
				consume(value, 0)
				continue
			}

			if prior.Type == "slash" && prior.Prev.Type != "bos" && len(rest) > 0 && rest[0] == '/' {
				end := ""
				if len(rest) > 1 {
					end = "|$"
				}

				state.Output = state.Output[:len(state.Output)-len(prior.Output+prev.Output)]
				prior.Output = "(?:" + prior.Output

				prev.Type = "globstar"
				prev.Output = globstarFn(opts.Dot) + slashLiteral + "|" + slashLiteral + end + ")"
				prev.Value += value

				state.Output += prior.Output + prev.Output
				state.Globstar = true

				consume(value+advance(), 0)

				push(&Token{Type: "slash", Value: "/", Output: ""})
				continue
			}

			if prior.Type == "bos" && len(rest) > 0 && rest[0] == '/' {
				prev.Type = "globstar"
				prev.Value += value
				prev.Output = "(?:^|" + slashLiteral + "|" + globstarFn(opts.Dot) + slashLiteral + ")"
				state.Output = prev.Output
				state.Globstar = true
				consume(value+advance(), 0)
				push(&Token{Type: "slash", Value: "/", Output: ""})
				continue
			}

			// Remove single star from output
			state.Output = state.Output[:len(state.Output)-len(prev.Output)]

			prev.Type = "globstar"
			prev.Output = globstarFn(opts.Dot)
			prev.Value += value

			state.Output += prev.Output
			state.Globstar = true
			consume(value, 0)
			continue
		}

		// Single star
		// JS source: parse.js lines 915-953
		tok := &Token{Type: "star", Value: value, Output: starStr}

		if opts.Bash {
			tok.Output = ".*?"
			if prev.Type == "bos" || prev.Type == "slash" {
				tok.Output = nodot + tok.Output
			}
			push(tok)
			continue
		}

		if prev != nil && (prev.Type == "bracket" || prev.Type == "paren") && opts.Regex == boolPtr(true) {
			tok.Output = value
			push(tok)
			continue
		}

		if state.Index == state.Start || prev.Type == "slash" || prev.Type == "dot" {
			if prev.Type == "dot" {
				state.Output += noDotSlash
				prev.Output += noDotSlash
			} else if opts.Dot {
				state.Output += noDotsSlash
				prev.Output += noDotsSlash
			} else {
				state.Output += nodot
				prev.Output += nodot
			}

			if peek(1) != "*" {
				state.Output += oneChar
				prev.Output += oneChar
			}
		}

		push(tok)
	}

	// --- Unclosed brackets/parens/braces ---
	// JS source: parse.js lines 955-971
	for state.Brackets > 0 {
		if opts.StrictBrackets {
			panic(syntaxError("closing", "]"))
		}
		state.Output = escapeLast(state.Output, '[', -1)
		decrement("brackets")
	}

	for state.Parens > 0 {
		if opts.StrictBrackets {
			panic(syntaxError("closing", ")"))
		}
		state.Output = escapeLast(state.Output, '(', -1)
		decrement("parens")
	}

	for state.Braces > 0 {
		if opts.StrictBrackets {
			panic(syntaxError("closing", "}"))
		}
		state.Output = escapeLast(state.Output, '{', -1)
		decrement("braces")
	}

	// Maybe trailing slash
	// JS source: parse.js lines 973-975
	if !opts.StrictSlashes && (prev.Type == "star" || prev.Type == "bracket") {
		push(&Token{Type: "maybe_slash", Value: "", Output: slashLiteral + "?"})
	}

	// Rebuild output if we had to backtrack
	// JS source: parse.js lines 978-988
	if state.Backtrack {
		state.Output = ""
		for _, tok := range state.Tokens {
			if tok.Output != "" {
				state.Output += tok.Output
			} else {
				state.Output += tok.Value
			}
			if tok.Suffix != "" {
				state.Output += tok.Suffix
			}
		}
	}

	state.Tokens = tokens
	return state
}

// Fastpaths generates optimized regex for common glob patterns.
// JS source: parse.js lines 999-1083
func Fastpaths(input string, opts *Options) string {
	if opts == nil {
		opts = &Options{}
	}

	maxLen := MaxLength
	if opts.MaxLength > 0 && opts.MaxLength < maxLen {
		maxLen = opts.MaxLength
	}
	if len(input) > maxLen {
		panic(fmt.Sprintf("Input length: %d, exceeds maximum allowed length: %d", len(input), maxLen))
	}

	if r, ok := REPLACEMENTS[input]; ok {
		input = r
	}

	pc := GlobChars(opts.Windows)
	dotLit := pc.DOT_LITERAL
	slashLit := pc.SLASH_LITERAL
	oneC := pc.ONE_CHAR
	dotsSl := pc.DOTS_SLASH
	noDotC := pc.NO_DOT
	noDotsC := pc.NO_DOTS
	noDotsSlC := pc.NO_DOTS_SLASH
	starC := pc.STAR
	startAnc := pc.START_ANCHOR

	capture := "?:"
	if opts.Capture {
		capture = ""
	}

	nodotC := noDotC
	if opts.Dot {
		nodotC = noDotsC
	}
	slashDot := noDotC
	if opts.Dot {
		slashDot = noDotsSlC
	}
	starLocal := starC
	if opts.Bash {
		starLocal = ".*?"
	}
	if opts.Capture {
		starLocal = "(" + starLocal + ")"
	}

	globstarLocal := func() string {
		if opts.Noglobstar {
			return starLocal
		}
		ds := dotLit
		if opts.Dot {
			ds = dotsSl
		}
		return "(" + capture + "(?:(?!" + startAnc + ds + ").)*?)"
	}

	var create func(str string) string
	create = func(str string) string {
		switch str {
		case "*":
			return nodotC + oneC + starLocal
		case ".*":
			return dotLit + oneC + starLocal
		case "*.*":
			return nodotC + starLocal + dotLit + oneC + starLocal
		case "*/*":
			return nodotC + starLocal + slashLit + oneC + slashDot + starLocal
		case "**":
			return nodotC + globstarLocal()
		case "**/*":
			return "(?:" + nodotC + globstarLocal() + slashLit + ")?" + slashDot + oneC + starLocal
		case "**/*.*":
			return "(?:" + nodotC + globstarLocal() + slashLit + ")?" + slashDot + starLocal + dotLit + oneC + starLocal
		case "**/.*":
			return "(?:" + nodotC + globstarLocal() + slashLit + ")?" + dotLit + oneC + starLocal
		default:
			// Try to match pattern like "prefix.ext"
			dotIdx := strings.LastIndex(str, ".")
			if dotIdx == -1 || dotIdx == 0 || dotIdx == len(str)-1 {
				return ""
			}
			// Check if ext is word chars only
			ext := str[dotIdx+1:]
			for _, c := range ext {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
					return ""
				}
			}
			source := create(str[:dotIdx])
			if source == "" {
				return ""
			}
			return source + dotLit + ext
		}
	}

	state := &ParseState{Negated: false, Prefix: ""}
	output := removePrefix(input, state)
	source := create(output)

	if source != "" && !opts.StrictSlashes {
		source += slashLit + "?"
	}

	return source
}

// --- Helpers ---

// matchStr tests if str matches a regex pattern.
func matchStr(pattern, str string) bool {
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return false
	}
	ok, _ := re.MatchString(str)
	return ok
}

// matchPrefix returns the prefix of str matching a regex pattern.
func matchPrefix(pattern, str string) string {
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return ""
	}
	m, _ := re.FindStringMatch(str)
	if m == nil {
		return ""
	}
	return m.String()
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}

// fastpathReplace applies the REGEX_SPECIAL_CHARS_BACKREF replacement logic.
// JS source: parse.js lines 278-305
// The JS regex is /(\\?)((\W)(\3*))/g
// Groups: $1=esc, $2=chars, $3=first, $4=rest
func fastpathReplace(input string, fn func(m, esc, chars, first, rest string, idx int) string) string {
	re, err := regexp2.Compile(regexSpecialCharsBackrefPattern, regexp2.None)
	if err != nil {
		return input
	}

	var result strings.Builder
	lastIdx := 0

	m, _ := re.FindStringMatch(input)
	for m != nil {
		groups := m.Groups()
		matchStart := m.Index
		matchStr := m.String()

		result.WriteString(input[lastIdx:matchStart])

		esc := ""
		chars := ""
		first := ""
		rest := ""
		if len(groups) > 1 {
			esc = groups[1].String()
		}
		if len(groups) > 2 {
			chars = groups[2].String()
		}
		if len(groups) > 3 {
			first = groups[3].String()
		}
		if len(groups) > 4 {
			rest = groups[4].String()
		}

		replacement := fn(matchStr, esc, chars, first, rest, matchStart)
		result.WriteString(replacement)
		lastIdx = matchStart + len(matchStr)

		m, _ = re.FindNextMatch(m)
	}

	result.WriteString(input[lastIdx:])
	return result.String()
}

// replaceBackslashRuns normalizes backslash runs.
// JS source: parse.js lines 311-314
func replaceBackslashRuns(output string) string {
	var result strings.Builder
	i := 0
	for i < len(output) {
		if output[i] == '\\' {
			// Count consecutive backslashes
			count := 0
			for i < len(output) && output[i] == '\\' {
				count++
				i++
			}
			if count%2 == 0 {
				result.WriteString(strings.Repeat(`\\`, count/2))
			} else {
				result.WriteString(strings.Repeat(`\\`, (count-1)/2))
				result.WriteByte('\\')
			}
		} else {
			result.WriteByte(output[i])
			i++
		}
	}
	return result.String()
}
