package secrets

// MaskSecret masks a secret value for logging and display.
// Shows first 4 and last 4 characters for secrets >12 chars.
// Returns "****" for short secrets.
func MaskSecret(value string) string {
	if len(value) <= 12 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
