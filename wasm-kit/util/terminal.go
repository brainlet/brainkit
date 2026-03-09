package util

const (
	ColorGray    = "\033[90m"
	ColorRed     = "\033[91m"
	ColorGreen   = "\033[92m"
	ColorYellow  = "\033[93m"
	ColorBlue    = "\033[94m"
	ColorMagenta = "\033[95m"
	ColorCyan    = "\033[96m"
	ColorWhite   = "\033[97m"
	ColorReset   = "\033[0m"
)

var colorsEnabled = true

func IsColorsEnabled() bool {
	return colorsEnabled
}

func SetColorsEnabled(enabled bool) bool {
	prev := colorsEnabled
	colorsEnabled = enabled
	return prev
}

func Colorize(text, color string) string {
	if colorsEnabled {
		return color + text + ColorReset
	}
	return text
}
