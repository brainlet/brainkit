//go:build darwin

package local

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// probeDevice asks macOS for the current output device name,
// volume (0-100), and mute state. Uses `osascript` + a
// `system_profiler` fallback so no external dep lands on the
// Go side.
func probeDevice(ctx context.Context) (device string, volume int, muted bool, err error) {
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Volume + mute via osascript. Returns two integers (volume
	// 0-100) + two bools ("true"/"false"). We only need output.
	out, oerr := exec.CommandContext(pctx, "osascript", "-e",
		`set v to get volume settings
		set volText to (output volume of v) as string
		set muteText to (output muted of v) as string
		return volText & "," & muteText`).Output()
	if oerr != nil {
		return "", -1, false, oerr
	}
	fields := strings.SplitN(strings.TrimSpace(string(out)), ",", 2)
	if len(fields) != 2 {
		return "", -1, false, errors.New("unexpected osascript output: " + string(out))
	}
	if v, perr := strconv.Atoi(strings.TrimSpace(fields[0])); perr == nil {
		volume = v
	} else {
		volume = -1
	}
	muted = strings.TrimSpace(fields[1]) == "true"

	// Output device name — SwitchAudioSource ships with brew;
	// fall back to system_profiler output device lookup which
	// is always present on macOS.
	if sw, serr := exec.CommandContext(pctx, "SwitchAudioSource", "-c", "-t", "output").Output(); serr == nil {
		device = strings.TrimSpace(string(sw))
	} else {
		sp, perr := exec.CommandContext(pctx, "system_profiler", "SPAudioDataType").Output()
		if perr == nil {
			// Look for the line that ends in "Default Output Device: Yes"
			// and grab the nearest preceding device name.
			lines := strings.Split(string(sp), "\n")
			var current string
			for _, line := range lines {
				trim := strings.TrimSpace(line)
				// Device names end with ':' and sit at a fixed
				// indent; naive detection is fine here.
				if strings.HasSuffix(trim, ":") && !strings.Contains(trim, " ") {
					continue
				}
				if strings.HasSuffix(trim, ":") && !strings.Contains(trim, "Device") {
					current = strings.TrimSuffix(trim, ":")
				}
				if strings.HasPrefix(trim, "Default Output Device: Yes") {
					device = current
					break
				}
			}
		}
	}
	return device, volume, muted, nil
}
