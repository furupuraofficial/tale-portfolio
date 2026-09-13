package audio

import (
	"fmt"
	"os/exec"
	"runtime"
)

type Player struct{}

func NewPlayer() *Player {
	return &Player{}
}

// PlayAudio plays an audio file using the system's default player
func (ap *Player) PlayAudio(filename string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("afplay", filename)
	case "linux":
		// Try multiple players
		if _, err := exec.LookPath("mpg123"); err == nil {
			cmd = exec.Command("mpg123", filename)
		} else if _, err := exec.LookPath("ffplay"); err == nil {
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", filename)
		} else if _, err := exec.LookPath("aplay"); err == nil {
			cmd = exec.Command("aplay", filename)
		} else {
			return fmt.Errorf("no audio player found (install mpg123, ffplay, or aplay)")
		}
	case "windows":
		// Use PowerShell to play audio on Windows
		cmd = exec.Command("powershell", "-c", fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync()", filename))
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	fmt.Printf("Playing audio: %s\n", filename)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to play audio: %w", err)
	}

	return nil
}
