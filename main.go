package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

//go:embed beep.mp3
var beepMP3 []byte

var audioContext *oto.Context

// version is set via ldflags at build time
var version = "dev"

// beepFunc is the function called to play a beep sound.
// It can be replaced in tests to prevent actual sound playback.
var beepFunc = playBeepImpl

func initAudio() error {
	// Decode the MP3 data to get audio format info
	reader := bytes.NewReader(beepMP3)
	decodedMP3, err := mp3.NewDecoder(reader)
	if err != nil {
		return fmt.Errorf("error decoding MP3: %w", err)
	}

	// Initialize oto context
	op := &oto.NewContextOptions{
		SampleRate:   decodedMP3.SampleRate(),
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return fmt.Errorf("error creating audio context: %w", err)
	}
	<-readyChan

	audioContext = ctx
	return nil
}

// playBeep calls beepFunc to play a beep sound.
// This indirection allows tests to replace beepFunc with a no-op.
func playBeep() {
	beepFunc()
}

// playBeepImpl is the actual implementation that plays the beep sound.
func playBeepImpl() {
	// Play the beep asynchronously so it doesn't block the timer
	go func() {
		// Decode the MP3 data each time (creates a fresh reader)
		reader := bytes.NewReader(beepMP3)
		decodedMP3, err := mp3.NewDecoder(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding MP3: %v\n", err)
			return
		}

		// Create a player and play the sound
		player := audioContext.NewPlayer(decodedMP3)
		player.Play()

		// Wait for the sound to finish
		for player.IsPlaying() {
			time.Sleep(10 * time.Millisecond)
		}

		// Clean up
		err = player.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing player: %v\n", err)
		}
	}()
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func parseIntList(s string) ([]int, error) {
	if s == "" {
		return []int{0}, nil
	}
	parts := strings.Split(s, ",")
	result := make([]int, len(parts))
	for i, part := range parts {
		val, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", part)
		}
		result[i] = val
	}
	return result, nil
}

// WaybarOutput represents JSON output for Waybar integration
type WaybarOutput struct {
	Text      string `json:"text"`
	Tooltip   string `json:"tooltip"`
	Class     string `json:"class"`
	Remaining int    `json:"remaining"`
}

// OutputMode represents the output format mode
type OutputMode int

const (
	ModeDefault OutputMode = iota
	ModeVerbose
	ModeJSON
	ModeWatch
)

// TimerState represents the current state of the timer
type TimerState struct {
	Intervals     []time.Duration
	MinutesList   []int
	SecondsList   []int
	IntervalIndex int
	BeepCount     int
	Paused        bool
	PausedAt      time.Duration
	NextBeep      time.Time
}

// NewTimerState creates a new timer state with the given intervals
func NewTimerState(intervals []time.Duration, minutesList, secondsList []int, startPaused bool) *TimerState {
	ts := &TimerState{
		Intervals:     intervals,
		MinutesList:   minutesList,
		SecondsList:   secondsList,
		IntervalIndex: 0,
		BeepCount:     0,
		Paused:        startPaused,
		PausedAt:      0,
	}
	if startPaused {
		ts.PausedAt = intervals[0]
	} else {
		ts.NextBeep = time.Now().Add(intervals[0])
	}
	return ts
}

// CurrentInterval returns the current interval duration
func (ts *TimerState) CurrentInterval() time.Duration {
	return ts.Intervals[ts.IntervalIndex]
}

// AdvanceInterval moves to the next interval in the rotation
func (ts *TimerState) AdvanceInterval() {
	ts.IntervalIndex = (ts.IntervalIndex + 1) % len(ts.Intervals)
}

// TogglePause toggles the pause state and returns the new pause state
func (ts *TimerState) TogglePause() bool {
	if ts.Paused {
		// Resume: set nextBeep based on remaining time
		ts.Paused = false
		ts.NextBeep = time.Now().Add(ts.PausedAt)
	} else {
		// Pause: save remaining time
		ts.Paused = true
		ts.PausedAt = time.Until(ts.NextBeep)
		if ts.PausedAt < 0 {
			ts.PausedAt = 0
		}
	}
	return ts.Paused
}

// Remaining returns the time remaining until next beep
func (ts *TimerState) Remaining() time.Duration {
	if ts.Paused {
		return ts.PausedAt
	}
	return time.Until(ts.NextBeep)
}

// TriggerBeep increments beep count, advances interval, and resets timer
func (ts *TimerState) TriggerBeep() {
	ts.BeepCount++
	ts.AdvanceInterval()
	ts.NextBeep = time.Now().Add(ts.CurrentInterval())
}

// ResetTimer resets the current interval without advancing
func (ts *TimerState) ResetTimer() {
	ts.NextBeep = time.Now().Add(ts.CurrentInterval())
}

// OutputConfig holds configuration for output formatting
type OutputConfig struct {
	Mode          OutputMode
	MinutesList   []int
	SecondsList   []int
	IntervalCount int
}

// FormatPausedOutput returns the output string for paused state
func FormatPausedOutput(config OutputConfig, pausedAt time.Duration) string {
	switch config.Mode {
	case ModeJSON:
		output := WaybarOutput{
			Text:      "Paused",
			Tooltip:   "Click to start",
			Class:     "paused",
			Remaining: int(pausedAt.Seconds()),
		}
		jsonBytes, _ := json.Marshal(output)
		return string(jsonBytes)
	case ModeWatch:
		return "PAUSED"
	case ModeVerbose:
		return fmt.Sprintf("\rPaused - %s remaining ", formatDuration(pausedAt))
	default:
		return ""
	}
}

// FormatTickOutput returns the output string for a timer tick
func FormatTickOutput(config OutputConfig, remaining time.Duration, intervalIndex int) string {
	remainingSecs := int(remaining.Round(time.Second).Seconds())
	switch config.Mode {
	case ModeJSON:
		var tooltip string
		if config.IntervalCount == 1 {
			tooltip = fmt.Sprintf("%dm %ds", config.MinutesList[0], config.SecondsList[0])
		} else {
			tooltip = fmt.Sprintf("Interval %d/%d: %dm %ds", intervalIndex+1, config.IntervalCount,
				config.MinutesList[intervalIndex], config.SecondsList[intervalIndex])
		}
		output := WaybarOutput{
			Text:      formatDuration(remaining),
			Tooltip:   tooltip,
			Class:     "counting",
			Remaining: remainingSecs,
		}
		jsonBytes, _ := json.Marshal(output)
		return string(jsonBytes)
	case ModeWatch:
		return formatDuration(remaining)
	case ModeVerbose:
		if config.IntervalCount == 1 {
			return fmt.Sprintf("\rNext beep in: %s ", formatDuration(remaining))
		}
		return fmt.Sprintf("\rNext beep in: %s (interval %d/%d: %dm %ds) ",
			formatDuration(remaining), intervalIndex+1, config.IntervalCount,
			config.MinutesList[intervalIndex], config.SecondsList[intervalIndex])
	default:
		return ""
	}
}

// FormatBeepOutput returns the output string for a beep event
func FormatBeepOutput(config OutputConfig, beepCount int, beepType string, intervalIndex int, timestamp time.Time) string {
	switch config.Mode {
	case ModeJSON:
		output := WaybarOutput{
			Text:      "BEEP",
			Tooltip:   fmt.Sprintf("Beep #%d (%s)", beepCount, beepType),
			Class:     "beep",
			Remaining: 0,
		}
		jsonBytes, _ := json.Marshal(output)
		return string(jsonBytes)
	case ModeWatch:
		return "BEEP"
	case ModeVerbose:
		if config.IntervalCount == 1 {
			return fmt.Sprintf("\r[%s] Beep #%d (%s)              \n", timestamp.Format("15:04:05"), beepCount, beepType)
		}
		return fmt.Sprintf("\r[%s] Beep #%d (%s) - next: %dm %ds     \n",
			timestamp.Format("15:04:05"), beepCount, beepType,
			config.MinutesList[intervalIndex], config.SecondsList[intervalIndex])
	default:
		return fmt.Sprintf("BEEP %s\n", timestamp.Format(time.RFC3339))
	}
}

// FormatResetOutput returns the output string for a timer reset
func FormatResetOutput(config OutputConfig, intervalIndex int, timestamp time.Time) string {
	if config.Mode != ModeVerbose {
		return ""
	}
	if config.IntervalCount == 1 {
		return fmt.Sprintf("\r[%s] Timer reset (silent)              \n", timestamp.Format("15:04:05"))
	}
	return fmt.Sprintf("\r[%s] Timer reset (silent) - interval %d/%d: %dm %ds      \n",
		timestamp.Format("15:04:05"), intervalIndex+1, config.IntervalCount,
		config.MinutesList[intervalIndex], config.SecondsList[intervalIndex])
}

// padLists ensures both lists have the same length by padding the shorter one
// with its last value. Returns the padded lists.
func padLists(minutesList, secondsList []int) ([]int, []int) {
	maxLen := len(minutesList)
	if len(secondsList) > maxLen {
		maxLen = len(secondsList)
	}

	// Create copies to avoid modifying original slices
	minutes := make([]int, len(minutesList))
	copy(minutes, minutesList)
	seconds := make([]int, len(secondsList))
	copy(seconds, secondsList)

	// Pad shorter list with its last value
	for len(minutes) < maxLen {
		minutes = append(minutes, minutes[len(minutes)-1])
	}
	for len(seconds) < maxLen {
		seconds = append(seconds, seconds[len(seconds)-1])
	}

	return minutes, seconds
}

// buildIntervals creates a list of time.Duration intervals from minutes and seconds lists.
// Returns an error if any interval would be zero or negative.
func buildIntervals(minutesList, secondsList []int) ([]time.Duration, error) {
	minutes, seconds := padLists(minutesList, secondsList)

	intervals := make([]time.Duration, len(minutes))
	for i := 0; i < len(minutes); i++ {
		totalSeconds := minutes[i]*60 + seconds[i]
		if totalSeconds <= 0 {
			return nil, fmt.Errorf("interval %d is %dm%ds (must be positive)", i+1, minutes[i], seconds[i])
		}
		intervals[i] = time.Duration(totalSeconds) * time.Second
	}

	return intervals, nil
}

func main() {
	minutesStr := flag.String("m", "0", "interval in minutes (comma-separated for multiple intervals)")
	secondsStr := flag.String("s", "0", "interval in seconds (comma-separated for multiple intervals)")
	verbose := flag.Bool("v", false, "verbose output (show countdown and status)")
	interactive := flag.Bool("i", false, "interactive mode (Enter to beep, Backspace to reset)")
	jsonMode := flag.Bool("json", false, "JSON output for Waybar integration")
	watchMode := flag.Bool("watch", false, "plain text countdown output")
	startPaused := flag.Bool("paused", false, "start in paused state (send SIGUSR1 to toggle)")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("go-interval version %s\n", version)
		os.Exit(0)
	}

	// Validate flag combinations
	if *jsonMode && *watchMode {
		fmt.Fprintf(os.Stderr, "Error: -json and -watch are mutually exclusive\n")
		os.Exit(1)
	}

	// Print PID for signal control (useful for Waybar on-click)
	if *startPaused || *jsonMode || *watchMode {
		fmt.Fprintf(os.Stderr, "PID: %d (send SIGUSR1 to toggle pause)\n", os.Getpid())
	}

	// Parse comma-separated values
	minutesList, err := parseIntList(*minutesStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing minutes: %v\n", err)
		os.Exit(1)
	}

	secondsList, err := parseIntList(*secondsStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing seconds: %v\n", err)
		os.Exit(1)
	}

	// Pad lists to equal length and build intervals
	minutesList, secondsList = padLists(minutesList, secondsList)

	intervals, err := buildIntervals(minutesList, secondsList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize audio system
	if err := initAudio(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing audio: %v\n", err)
		os.Exit(1)
	}

	// Verbose mode: show banner and instructions
	if *verbose {
		fmt.Printf("=== Interval Beeper ===\n")
		if len(intervals) == 1 {
			fmt.Printf("Beeping every %d minutes %d seconds.\n", minutesList[0], secondsList[0])
		} else {
			fmt.Printf("Beeping with rotating intervals:\n")
			for i := 0; i < len(intervals); i++ {
				fmt.Printf("  %d. %dm %ds\n", i+1, minutesList[i], secondsList[i])
			}
		}
		if *interactive {
			fmt.Printf("Press Enter to beep immediately and reset timer.\n")
			fmt.Printf("Press Backspace to reset timer silently. Press Ctrl+C to stop.\n\n")
		} else {
			fmt.Printf("Press Ctrl+C to stop.\n\n")
		}
	}

	// Channels to signal key presses (only used in interactive mode)
	enterPressed := make(chan bool)
	backspacePressed := make(chan bool)

	// Goroutine to listen for key presses (only in interactive mode)
	if *interactive {
		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				b, err := reader.ReadByte()
				if err != nil {
					continue
				}
				if b == '\n' {
					enterPressed <- true
				} else if b == 127 || b == 8 { // 127 is DEL, 8 is backspace
					backspacePressed <- true
				}
			}
		}()
	}

	beepCount := 0
	intervalIndex := 0
	currentInterval := intervals[intervalIndex]
	nextBeep := time.Now().Add(currentInterval)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Pause state
	paused := *startPaused
	pausedAt := time.Duration(0) // remaining time when paused

	// Signal handling for SIGUSR1 (toggle pause)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1)

	// Helper function to output paused state
	outputPaused := func() {
		if *jsonMode {
			output := WaybarOutput{
				Text:      "Paused",
				Tooltip:   "Click to start",
				Class:     "paused",
				Remaining: int(pausedAt.Seconds()),
			}
			jsonBytes, _ := json.Marshal(output)
			fmt.Println(string(jsonBytes))
		} else if *watchMode {
			fmt.Println("PAUSED")
		} else if *verbose {
			fmt.Printf("\rPaused - %s remaining ", formatDuration(pausedAt))
			os.Stdout.Sync()
		}
	}

	// Helper function to output tick based on mode
	outputTick := func(remaining time.Duration) {
		remainingSecs := int(remaining.Round(time.Second).Seconds())
		if *jsonMode {
			var tooltip string
			if len(intervals) == 1 {
				tooltip = fmt.Sprintf("%dm %ds", minutesList[0], secondsList[0])
			} else {
				tooltip = fmt.Sprintf("Interval %d/%d: %dm %ds", intervalIndex+1, len(intervals),
					minutesList[intervalIndex], secondsList[intervalIndex])
			}
			output := WaybarOutput{
				Text:      formatDuration(remaining),
				Tooltip:   tooltip,
				Class:     "counting",
				Remaining: remainingSecs,
			}
			jsonBytes, _ := json.Marshal(output)
			fmt.Println(string(jsonBytes))
		} else if *watchMode {
			fmt.Println(formatDuration(remaining))
		} else if *verbose {
			if len(intervals) == 1 {
				fmt.Printf("\rNext beep in: %s ", formatDuration(remaining))
			} else {
				fmt.Printf("\rNext beep in: %s (interval %d/%d: %dm %ds) ",
					formatDuration(remaining), intervalIndex+1, len(intervals),
					minutesList[intervalIndex], secondsList[intervalIndex])
			}
			os.Stdout.Sync()
		}
		// Default mode: no tick output
	}

	// Helper function to output beep based on mode
	outputBeep := func(beepType string) {
		if *jsonMode {
			output := WaybarOutput{
				Text:      "BEEP",
				Tooltip:   fmt.Sprintf("Beep #%d (%s)", beepCount, beepType),
				Class:     "beep",
				Remaining: 0,
			}
			jsonBytes, _ := json.Marshal(output)
			fmt.Println(string(jsonBytes))
		} else if *watchMode {
			fmt.Println("BEEP")
		} else if *verbose {
			if len(intervals) == 1 {
				fmt.Printf("\r[%s] Beep #%d (%s)              \n", time.Now().Format("15:04:05"), beepCount, beepType)
			} else {
				fmt.Printf("\r[%s] Beep #%d (%s) - next: %dm %ds     \n",
					time.Now().Format("15:04:05"), beepCount, beepType,
					minutesList[intervalIndex], secondsList[intervalIndex])
			}
			os.Stdout.Sync()
		} else {
			// Default mode: simple beep line
			fmt.Printf("BEEP %s\n", time.Now().Format(time.RFC3339))
		}
	}

	// Helper function to output reset (verbose only)
	outputReset := func() {
		if *verbose {
			if len(intervals) == 1 {
				fmt.Printf("\r[%s] Timer reset (silent)              \n", time.Now().Format("15:04:05"))
			} else {
				fmt.Printf("\r[%s] Timer reset (silent) - interval %d/%d: %dm %ds      \n",
					time.Now().Format("15:04:05"), intervalIndex+1, len(intervals),
					minutesList[intervalIndex], secondsList[intervalIndex])
			}
			os.Stdout.Sync()
		}
		// Other modes: silent reset is truly silent
	}

	// If starting paused, set initial pausedAt
	if paused {
		pausedAt = currentInterval
		outputPaused()
	}

	for {
		select {
		case <-sigChan:
			// Toggle pause state
			if paused {
				// Resume: set nextBeep based on remaining time
				paused = false
				nextBeep = time.Now().Add(pausedAt)
				if *verbose {
					fmt.Printf("\r[%s] Resumed                           \n", time.Now().Format("15:04:05"))
					os.Stdout.Sync()
				}
			} else {
				// Pause: save remaining time
				paused = true
				pausedAt = time.Until(nextBeep)
				if pausedAt < 0 {
					pausedAt = 0
				}
				if *verbose {
					fmt.Printf("\r[%s] Paused                            \n", time.Now().Format("15:04:05"))
					os.Stdout.Sync()
				}
				outputPaused()
			}

		case <-ticker.C:
			if paused {
				outputPaused()
				continue
			}

			remaining := time.Until(nextBeep)

			if remaining <= 0 {
				beepCount++
				playBeep()

				// Move to next interval in the rotation
				intervalIndex = (intervalIndex + 1) % len(intervals)
				currentInterval = intervals[intervalIndex]

				outputBeep("automatic")
				nextBeep = time.Now().Add(currentInterval)
			} else {
				outputTick(remaining)
			}

		case <-enterPressed:
			if paused {
				continue
			}
			beepCount++
			playBeep()

			// Move to next interval in the rotation
			intervalIndex = (intervalIndex + 1) % len(intervals)
			currentInterval = intervals[intervalIndex]

			outputBeep("manual")
			nextBeep = time.Now().Add(currentInterval)

		case <-backspacePressed:
			if paused {
				continue
			}
			outputReset()
			nextBeep = time.Now().Add(currentInterval)
		}
	}
}
