// Ported from: packages/ai/src/util/fix-json.ts
package util

import "strings"

// fixJSONState represents the state of the JSON scanner.
type fixJSONState int

const (
	stateRoot fixJSONState = iota
	stateFinish
	stateInsideString
	stateInsideStringEscape
	stateInsideLiteral
	stateInsideNumber
	stateInsideObjectStart
	stateInsideObjectKey
	stateInsideObjectAfterKey
	stateInsideObjectBeforeValue
	stateInsideObjectAfterValue
	stateInsideObjectAfterComma
	stateInsideArrayStart
	stateInsideArrayAfterValue
	stateInsideArrayAfterComma
)

// FixJSON attempts to repair partial/incomplete JSON strings by completing
// open structures. It performs a single linear time scan pass over the
// partial JSON.
func FixJSON(input string) string {
	stack := []fixJSONState{stateRoot}
	lastValidIndex := -1
	literalStart := -1

	processValueStart := func(char byte, i int, swapState fixJSONState) {
		switch char {
		case '"':
			lastValidIndex = i
			stack = stack[:len(stack)-1]
			stack = append(stack, swapState)
			stack = append(stack, stateInsideString)

		case 'f', 't', 'n':
			lastValidIndex = i
			literalStart = i
			stack = stack[:len(stack)-1]
			stack = append(stack, swapState)
			stack = append(stack, stateInsideLiteral)

		case '-':
			stack = stack[:len(stack)-1]
			stack = append(stack, swapState)
			stack = append(stack, stateInsideNumber)

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			lastValidIndex = i
			stack = stack[:len(stack)-1]
			stack = append(stack, swapState)
			stack = append(stack, stateInsideNumber)

		case '{':
			lastValidIndex = i
			stack = stack[:len(stack)-1]
			stack = append(stack, swapState)
			stack = append(stack, stateInsideObjectStart)

		case '[':
			lastValidIndex = i
			stack = stack[:len(stack)-1]
			stack = append(stack, swapState)
			stack = append(stack, stateInsideArrayStart)
		}
	}

	processAfterObjectValue := func(char byte, i int) {
		switch char {
		case ',':
			stack = stack[:len(stack)-1]
			stack = append(stack, stateInsideObjectAfterComma)
		case '}':
			lastValidIndex = i
			stack = stack[:len(stack)-1]
		}
	}

	processAfterArrayValue := func(char byte, i int) {
		switch char {
		case ',':
			stack = stack[:len(stack)-1]
			stack = append(stack, stateInsideArrayAfterComma)
		case ']':
			lastValidIndex = i
			stack = stack[:len(stack)-1]
		}
	}

	for i := 0; i < len(input); i++ {
		char := input[i]
		currentState := stack[len(stack)-1]

		switch currentState {
		case stateRoot:
			processValueStart(char, i, stateFinish)

		case stateInsideObjectStart:
			switch char {
			case '"':
				stack = stack[:len(stack)-1]
				stack = append(stack, stateInsideObjectKey)
			case '}':
				lastValidIndex = i
				stack = stack[:len(stack)-1]
			}

		case stateInsideObjectAfterComma:
			switch char {
			case '"':
				stack = stack[:len(stack)-1]
				stack = append(stack, stateInsideObjectKey)
			}

		case stateInsideObjectKey:
			switch char {
			case '"':
				stack = stack[:len(stack)-1]
				stack = append(stack, stateInsideObjectAfterKey)
			}

		case stateInsideObjectAfterKey:
			switch char {
			case ':':
				stack = stack[:len(stack)-1]
				stack = append(stack, stateInsideObjectBeforeValue)
			}

		case stateInsideObjectBeforeValue:
			processValueStart(char, i, stateInsideObjectAfterValue)

		case stateInsideObjectAfterValue:
			processAfterObjectValue(char, i)

		case stateInsideString:
			switch char {
			case '"':
				stack = stack[:len(stack)-1]
				lastValidIndex = i
			case '\\':
				stack = append(stack, stateInsideStringEscape)
			default:
				lastValidIndex = i
			}

		case stateInsideArrayStart:
			switch char {
			case ']':
				lastValidIndex = i
				stack = stack[:len(stack)-1]
			default:
				lastValidIndex = i
				processValueStart(char, i, stateInsideArrayAfterValue)
			}

		case stateInsideArrayAfterValue:
			switch char {
			case ',':
				stack = stack[:len(stack)-1]
				stack = append(stack, stateInsideArrayAfterComma)
			case ']':
				lastValidIndex = i
				stack = stack[:len(stack)-1]
			default:
				lastValidIndex = i
			}

		case stateInsideArrayAfterComma:
			processValueStart(char, i, stateInsideArrayAfterValue)

		case stateInsideStringEscape:
			stack = stack[:len(stack)-1]
			lastValidIndex = i

		case stateInsideNumber:
			switch char {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				lastValidIndex = i

			case 'e', 'E', '-', '.':
				// continue number parsing

			case ',':
				stack = stack[:len(stack)-1]
				if stack[len(stack)-1] == stateInsideArrayAfterValue {
					processAfterArrayValue(char, i)
				}
				if stack[len(stack)-1] == stateInsideObjectAfterValue {
					processAfterObjectValue(char, i)
				}

			case '}':
				stack = stack[:len(stack)-1]
				if stack[len(stack)-1] == stateInsideObjectAfterValue {
					processAfterObjectValue(char, i)
				}

			case ']':
				stack = stack[:len(stack)-1]
				if stack[len(stack)-1] == stateInsideArrayAfterValue {
					processAfterArrayValue(char, i)
				}

			default:
				stack = stack[:len(stack)-1]
			}

		case stateInsideLiteral:
			partialLiteral := input[literalStart : i+1]
			if !strings.HasPrefix("false", partialLiteral) &&
				!strings.HasPrefix("true", partialLiteral) &&
				!strings.HasPrefix("null", partialLiteral) {
				stack = stack[:len(stack)-1]
				if stack[len(stack)-1] == stateInsideObjectAfterValue {
					processAfterObjectValue(char, i)
				} else if stack[len(stack)-1] == stateInsideArrayAfterValue {
					processAfterArrayValue(char, i)
				}
			} else {
				lastValidIndex = i
			}
		}
	}

	result := ""
	if lastValidIndex >= 0 {
		result = input[:lastValidIndex+1]
	}

	for i := len(stack) - 1; i >= 0; i-- {
		state := stack[i]
		switch state {
		case stateInsideString:
			result += `"`

		case stateInsideObjectKey,
			stateInsideObjectAfterKey,
			stateInsideObjectAfterComma,
			stateInsideObjectStart,
			stateInsideObjectBeforeValue,
			stateInsideObjectAfterValue:
			result += "}"

		case stateInsideArrayStart,
			stateInsideArrayAfterComma,
			stateInsideArrayAfterValue:
			result += "]"

		case stateInsideLiteral:
			partialLiteral := input[literalStart:]
			if strings.HasPrefix("true", partialLiteral) {
				result += "true"[len(partialLiteral):]
			} else if strings.HasPrefix("false", partialLiteral) {
				result += "false"[len(partialLiteral):]
			} else if strings.HasPrefix("null", partialLiteral) {
				result += "null"[len(partialLiteral):]
			}
		}
	}

	return result
}
