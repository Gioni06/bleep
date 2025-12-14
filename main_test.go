package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMain sets up test environment to prevent sound playback
func TestMain(m *testing.M) {
	// Replace beepFunc with a no-op to prevent sound during tests
	beepFunc = func() {}
	m.Run()
}

// TestFormatDuration tests the formatDuration function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero seconds",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "only seconds",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "one minute",
			duration: 60 * time.Second,
			expected: "1m 0s",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			expected: "5m 30s",
		},
		{
			name:     "large duration",
			duration: 25*time.Minute + 45*time.Second,
			expected: "25m 45s",
		},
		{
			name:     "rounds to nearest second",
			duration: 5*time.Minute + 30*time.Second + 500*time.Millisecond,
			expected: "5m 31s",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			expected: "59s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestParseIntList tests the parseIntList function
func TestParseIntList(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []int
		expectError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []int{0},
		},
		{
			name:     "single value",
			input:    "25",
			expected: []int{25},
		},
		{
			name:     "multiple values",
			input:    "25,5,10",
			expected: []int{25, 5, 10},
		},
		{
			name:     "values with spaces",
			input:    "25, 5, 10",
			expected: []int{25, 5, 10},
		},
		{
			name:     "zero value",
			input:    "0",
			expected: []int{0},
		},
		{
			name:        "invalid number",
			input:       "25,abc",
			expectError: true,
		},
		{
			name:        "float not allowed",
			input:       "25.5",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIntList(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseIntList(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseIntList(%q) unexpected error: %v", tt.input, err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("parseIntList(%q) = %v (len %d), want %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseIntList(%q)[%d] = %d, want %d", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestPadLists tests the padLists function
func TestPadLists(t *testing.T) {
	tests := []struct {
		name            string
		minutes         []int
		seconds         []int
		expectedMinutes []int
		expectedSeconds []int
	}{
		{
			name:            "equal length",
			minutes:         []int{25, 5},
			seconds:         []int{0, 30},
			expectedMinutes: []int{25, 5},
			expectedSeconds: []int{0, 30},
		},
		{
			name:            "minutes longer",
			minutes:         []int{25, 5, 10},
			seconds:         []int{30},
			expectedMinutes: []int{25, 5, 10},
			expectedSeconds: []int{30, 30, 30},
		},
		{
			name:            "seconds longer",
			minutes:         []int{1},
			seconds:         []int{30, 45, 15},
			expectedMinutes: []int{1, 1, 1},
			expectedSeconds: []int{30, 45, 15},
		},
		{
			name:            "single values",
			minutes:         []int{25},
			seconds:         []int{0},
			expectedMinutes: []int{25},
			expectedSeconds: []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultMinutes, resultSeconds := padLists(tt.minutes, tt.seconds)

			if len(resultMinutes) != len(tt.expectedMinutes) {
				t.Errorf("padLists minutes length = %d, want %d", len(resultMinutes), len(tt.expectedMinutes))
			}
			if len(resultSeconds) != len(tt.expectedSeconds) {
				t.Errorf("padLists seconds length = %d, want %d", len(resultSeconds), len(tt.expectedSeconds))
			}

			for i, v := range resultMinutes {
				if v != tt.expectedMinutes[i] {
					t.Errorf("padLists minutes[%d] = %d, want %d", i, v, tt.expectedMinutes[i])
				}
			}
			for i, v := range resultSeconds {
				if v != tt.expectedSeconds[i] {
					t.Errorf("padLists seconds[%d] = %d, want %d", i, v, tt.expectedSeconds[i])
				}
			}
		})
	}
}

// TestPadListsDoesNotModifyOriginal tests that padLists doesn't modify original slices
func TestPadListsDoesNotModifyOriginal(t *testing.T) {
	minutes := []int{25}
	seconds := []int{30, 45}

	originalMinutes := make([]int, len(minutes))
	copy(originalMinutes, minutes)

	padLists(minutes, seconds)

	if len(minutes) != len(originalMinutes) {
		t.Errorf("original minutes slice was modified: got len %d, want %d", len(minutes), len(originalMinutes))
	}
}

// TestBuildIntervals tests the buildIntervals function
func TestBuildIntervals(t *testing.T) {
	tests := []struct {
		name        string
		minutes     []int
		seconds     []int
		expected    []time.Duration
		expectError bool
	}{
		{
			name:     "single interval in minutes",
			minutes:  []int{25},
			seconds:  []int{0},
			expected: []time.Duration{25 * time.Minute},
		},
		{
			name:     "single interval in seconds",
			minutes:  []int{0},
			seconds:  []int{45},
			expected: []time.Duration{45 * time.Second},
		},
		{
			name:    "mixed minutes and seconds",
			minutes: []int{1},
			seconds: []int{30},
			expected: []time.Duration{
				1*time.Minute + 30*time.Second,
			},
		},
		{
			name:    "multiple intervals",
			minutes: []int{25, 5},
			seconds: []int{0, 0},
			expected: []time.Duration{
				25 * time.Minute,
				5 * time.Minute,
			},
		},
		{
			name:    "pomodoro style",
			minutes: []int{25, 5, 25, 15},
			seconds: []int{0, 0, 0, 0},
			expected: []time.Duration{
				25 * time.Minute,
				5 * time.Minute,
				25 * time.Minute,
				15 * time.Minute,
			},
		},
		{
			name:        "zero interval error",
			minutes:     []int{0},
			seconds:     []int{0},
			expectError: true,
		},
		{
			name:        "one valid one invalid",
			minutes:     []int{25, 0},
			seconds:     []int{0, 0},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildIntervals(tt.minutes, tt.seconds)
			if tt.expectError {
				if err == nil {
					t.Errorf("buildIntervals expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("buildIntervals unexpected error: %v", err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("buildIntervals length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("buildIntervals[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestBuildIntervalsWithPadding tests that buildIntervals correctly pads lists
func TestBuildIntervalsWithPadding(t *testing.T) {
	// minutes has 1 value, seconds has 3
	// should pad minutes to [1, 1, 1]
	minutes := []int{1}
	seconds := []int{30, 45, 15}

	result, err := buildIntervals(minutes, seconds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Duration{
		1*time.Minute + 30*time.Second,
		1*time.Minute + 45*time.Second,
		1*time.Minute + 15*time.Second,
	}

	if len(result) != len(expected) {
		t.Errorf("buildIntervals length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("buildIntervals[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestWaybarOutputJSON tests JSON marshaling of WaybarOutput
func TestWaybarOutputJSON(t *testing.T) {
	tests := []struct {
		name     string
		output   WaybarOutput
		expected string
	}{
		{
			name: "counting state",
			output: WaybarOutput{
				Text:      "24m 35s",
				Tooltip:   "25m 0s",
				Class:     "counting",
				Remaining: 1475,
			},
			expected: `{"text":"24m 35s","tooltip":"25m 0s","class":"counting","remaining":1475}`,
		},
		{
			name: "paused state",
			output: WaybarOutput{
				Text:      "Paused",
				Tooltip:   "Click to start",
				Class:     "paused",
				Remaining: 1500,
			},
			expected: `{"text":"Paused","tooltip":"Click to start","class":"paused","remaining":1500}`,
		},
		{
			name: "beep state",
			output: WaybarOutput{
				Text:      "BEEP",
				Tooltip:   "Beep #1 (automatic)",
				Class:     "beep",
				Remaining: 0,
			},
			expected: `{"text":"BEEP","tooltip":"Beep #1 (automatic)","class":"beep","remaining":0}`,
		},
		{
			name: "multi-interval tooltip",
			output: WaybarOutput{
				Text:      "4m 30s",
				Tooltip:   "Interval 2/3: 5m 0s",
				Class:     "counting",
				Remaining: 270,
			},
			expected: `{"text":"4m 30s","tooltip":"Interval 2/3: 5m 0s","class":"counting","remaining":270}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("json.Marshal error: %v", err)
			}
			result := string(jsonBytes)
			if result != tt.expected {
				t.Errorf("JSON output = %s, want %s", result, tt.expected)
			}
		})
	}
}

// TestWaybarOutputUnmarshal tests JSON unmarshaling of WaybarOutput
func TestWaybarOutputUnmarshal(t *testing.T) {
	input := `{"text":"24m 35s","tooltip":"25m 0s","class":"counting","remaining":1475}`

	var output WaybarOutput
	err := json.Unmarshal([]byte(input), &output)
	if err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if output.Text != "24m 35s" {
		t.Errorf("Text = %q, want %q", output.Text, "24m 35s")
	}
	if output.Tooltip != "25m 0s" {
		t.Errorf("Tooltip = %q, want %q", output.Tooltip, "25m 0s")
	}
	if output.Class != "counting" {
		t.Errorf("Class = %q, want %q", output.Class, "counting")
	}
	if output.Remaining != 1475 {
		t.Errorf("Remaining = %d, want %d", output.Remaining, 1475)
	}
}

// TestPlayBeepCallsBeepFunc tests that playBeep calls beepFunc
func TestPlayBeepCallsBeepFunc(t *testing.T) {
	called := false
	originalBeepFunc := beepFunc
	defer func() { beepFunc = originalBeepFunc }()

	beepFunc = func() {
		called = true
	}

	playBeep()

	if !called {
		t.Error("playBeep() did not call beepFunc")
	}
}

// TestNewTimerState tests the NewTimerState constructor
func TestNewTimerState(t *testing.T) {
	intervals := []time.Duration{25 * time.Minute, 5 * time.Minute}
	minutes := []int{25, 5}
	seconds := []int{0, 0}

	t.Run("starts running", func(t *testing.T) {
		ts := NewTimerState(intervals, minutes, seconds, false)

		if ts.Paused {
			t.Error("expected timer to start running, got paused")
		}
		if ts.IntervalIndex != 0 {
			t.Errorf("IntervalIndex = %d, want 0", ts.IntervalIndex)
		}
		if ts.BeepCount != 0 {
			t.Errorf("BeepCount = %d, want 0", ts.BeepCount)
		}
		if ts.NextBeep.IsZero() {
			t.Error("NextBeep should not be zero when starting running")
		}
	})

	t.Run("starts paused", func(t *testing.T) {
		ts := NewTimerState(intervals, minutes, seconds, true)

		if !ts.Paused {
			t.Error("expected timer to start paused")
		}
		if ts.PausedAt != intervals[0] {
			t.Errorf("PausedAt = %v, want %v", ts.PausedAt, intervals[0])
		}
	})
}

// TestTimerStateCurrentInterval tests the CurrentInterval method
func TestTimerStateCurrentInterval(t *testing.T) {
	intervals := []time.Duration{25 * time.Minute, 5 * time.Minute, 10 * time.Minute}
	ts := NewTimerState(intervals, []int{25, 5, 10}, []int{0, 0, 0}, false)

	if ts.CurrentInterval() != 25*time.Minute {
		t.Errorf("CurrentInterval() = %v, want %v", ts.CurrentInterval(), 25*time.Minute)
	}

	ts.IntervalIndex = 1
	if ts.CurrentInterval() != 5*time.Minute {
		t.Errorf("CurrentInterval() = %v, want %v", ts.CurrentInterval(), 5*time.Minute)
	}

	ts.IntervalIndex = 2
	if ts.CurrentInterval() != 10*time.Minute {
		t.Errorf("CurrentInterval() = %v, want %v", ts.CurrentInterval(), 10*time.Minute)
	}
}

// TestTimerStateAdvanceInterval tests the AdvanceInterval method
func TestTimerStateAdvanceInterval(t *testing.T) {
	intervals := []time.Duration{25 * time.Minute, 5 * time.Minute}
	ts := NewTimerState(intervals, []int{25, 5}, []int{0, 0}, false)

	ts.AdvanceInterval()
	if ts.IntervalIndex != 1 {
		t.Errorf("IntervalIndex = %d, want 1", ts.IntervalIndex)
	}

	ts.AdvanceInterval()
	if ts.IntervalIndex != 0 {
		t.Errorf("IntervalIndex = %d, want 0 (should wrap around)", ts.IntervalIndex)
	}
}

// TestTimerStateTogglePause tests the TogglePause method
func TestTimerStateTogglePause(t *testing.T) {
	intervals := []time.Duration{25 * time.Minute}
	ts := NewTimerState(intervals, []int{25}, []int{0}, false)

	// Should start running
	if ts.Paused {
		t.Error("expected timer to start running")
	}

	// Toggle to paused
	paused := ts.TogglePause()
	if !paused {
		t.Error("TogglePause should return true when pausing")
	}
	if !ts.Paused {
		t.Error("timer should be paused after toggle")
	}
	if ts.PausedAt <= 0 {
		t.Error("PausedAt should be positive when paused")
	}

	// Toggle back to running
	paused = ts.TogglePause()
	if paused {
		t.Error("TogglePause should return false when resuming")
	}
	if ts.Paused {
		t.Error("timer should be running after second toggle")
	}
}

// TestTimerStateTriggerBeep tests the TriggerBeep method
func TestTimerStateTriggerBeep(t *testing.T) {
	intervals := []time.Duration{25 * time.Minute, 5 * time.Minute}
	ts := NewTimerState(intervals, []int{25, 5}, []int{0, 0}, false)

	initialBeepCount := ts.BeepCount
	ts.TriggerBeep()

	if ts.BeepCount != initialBeepCount+1 {
		t.Errorf("BeepCount = %d, want %d", ts.BeepCount, initialBeepCount+1)
	}
	if ts.IntervalIndex != 1 {
		t.Errorf("IntervalIndex = %d, want 1", ts.IntervalIndex)
	}
}

// TestTimerStateResetTimer tests the ResetTimer method
func TestTimerStateResetTimer(t *testing.T) {
	intervals := []time.Duration{25 * time.Minute}
	ts := NewTimerState(intervals, []int{25}, []int{0}, false)

	// Wait a tiny bit and then reset
	time.Sleep(10 * time.Millisecond)
	beforeReset := time.Now()
	ts.ResetTimer()

	if ts.NextBeep.Before(beforeReset) {
		t.Error("NextBeep should be after reset time")
	}
}

// TestTimerStateRemaining tests the Remaining method
func TestTimerStateRemaining(t *testing.T) {
	intervals := []time.Duration{1 * time.Minute}
	ts := NewTimerState(intervals, []int{1}, []int{0}, false)

	remaining := ts.Remaining()
	if remaining <= 0 || remaining > 1*time.Minute {
		t.Errorf("Remaining() = %v, expected between 0 and 1 minute", remaining)
	}

	// Test when paused
	ts.Paused = true
	ts.PausedAt = 30 * time.Second
	if ts.Remaining() != 30*time.Second {
		t.Errorf("Remaining() when paused = %v, want 30s", ts.Remaining())
	}
}

// TestFormatPausedOutput tests the FormatPausedOutput function
func TestFormatPausedOutput(t *testing.T) {
	config := OutputConfig{
		MinutesList:   []int{25},
		SecondsList:   []int{0},
		IntervalCount: 1,
	}

	tests := []struct {
		name     string
		mode     OutputMode
		pausedAt time.Duration
		contains string
	}{
		{
			name:     "JSON mode",
			mode:     ModeJSON,
			pausedAt: 25 * time.Minute,
			contains: `"class":"paused"`,
		},
		{
			name:     "Watch mode",
			mode:     ModeWatch,
			pausedAt: 25 * time.Minute,
			contains: "PAUSED",
		},
		{
			name:     "Verbose mode",
			mode:     ModeVerbose,
			pausedAt: 25 * time.Minute,
			contains: "Paused",
		},
		{
			name:     "Default mode returns empty",
			mode:     ModeDefault,
			pausedAt: 25 * time.Minute,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Mode = tt.mode
			result := FormatPausedOutput(config, tt.pausedAt)
			if tt.contains == "" && result != "" {
				t.Errorf("expected empty string, got %q", result)
			} else if tt.contains != "" && !containsString(result, tt.contains) {
				t.Errorf("expected output to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

// TestFormatTickOutput tests the FormatTickOutput function
func TestFormatTickOutput(t *testing.T) {
	t.Run("JSON mode single interval", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeJSON,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatTickOutput(config, 24*time.Minute+35*time.Second, 0)
		if !containsString(result, `"class":"counting"`) {
			t.Errorf("expected counting class, got %s", result)
		}
		if !containsString(result, `"text":"24m 35s"`) {
			t.Errorf("expected text with time, got %s", result)
		}
	})

	t.Run("JSON mode multiple intervals", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeJSON,
			MinutesList:   []int{25, 5},
			SecondsList:   []int{0, 0},
			IntervalCount: 2,
		}
		result := FormatTickOutput(config, 4*time.Minute+30*time.Second, 1)
		if !containsString(result, "Interval 2/2") {
			t.Errorf("expected interval info in tooltip, got %s", result)
		}
	})

	t.Run("Watch mode", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeWatch,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatTickOutput(config, 24*time.Minute+35*time.Second, 0)
		if result != "24m 35s" {
			t.Errorf("expected '24m 35s', got %q", result)
		}
	})

	t.Run("Verbose mode single interval", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeVerbose,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatTickOutput(config, 24*time.Minute+35*time.Second, 0)
		if !containsString(result, "Next beep in:") {
			t.Errorf("expected 'Next beep in:', got %q", result)
		}
	})

	t.Run("Default mode returns empty", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeDefault,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatTickOutput(config, 24*time.Minute+35*time.Second, 0)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

// TestFormatBeepOutput tests the FormatBeepOutput function
func TestFormatBeepOutput(t *testing.T) {
	timestamp := time.Date(2024, 12, 13, 15, 30, 0, 0, time.Local)

	t.Run("JSON mode", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeJSON,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatBeepOutput(config, 1, "automatic", 0, timestamp)
		if !containsString(result, `"class":"beep"`) {
			t.Errorf("expected beep class, got %s", result)
		}
		if !containsString(result, `"text":"BEEP"`) {
			t.Errorf("expected BEEP text, got %s", result)
		}
	})

	t.Run("Watch mode", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeWatch,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatBeepOutput(config, 1, "automatic", 0, timestamp)
		if result != "BEEP" {
			t.Errorf("expected 'BEEP', got %q", result)
		}
	})

	t.Run("Verbose mode single interval", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeVerbose,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatBeepOutput(config, 1, "automatic", 0, timestamp)
		if !containsString(result, "Beep #1") {
			t.Errorf("expected 'Beep #1', got %q", result)
		}
		if !containsString(result, "automatic") {
			t.Errorf("expected 'automatic', got %q", result)
		}
	})

	t.Run("Verbose mode multiple intervals", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeVerbose,
			MinutesList:   []int{25, 5},
			SecondsList:   []int{0, 0},
			IntervalCount: 2,
		}
		result := FormatBeepOutput(config, 2, "manual", 1, timestamp)
		if !containsString(result, "Beep #2") {
			t.Errorf("expected 'Beep #2', got %q", result)
		}
		if !containsString(result, "next:") {
			t.Errorf("expected 'next:', got %q", result)
		}
	})

	t.Run("Default mode", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeDefault,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatBeepOutput(config, 1, "automatic", 0, timestamp)
		if !containsString(result, "BEEP") {
			t.Errorf("expected 'BEEP', got %q", result)
		}
	})
}

// TestFormatResetOutput tests the FormatResetOutput function
func TestFormatResetOutput(t *testing.T) {
	timestamp := time.Date(2024, 12, 13, 15, 30, 0, 0, time.Local)

	t.Run("Verbose mode single interval", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeVerbose,
			MinutesList:   []int{25},
			SecondsList:   []int{0},
			IntervalCount: 1,
		}
		result := FormatResetOutput(config, 0, timestamp)
		if !containsString(result, "Timer reset") {
			t.Errorf("expected 'Timer reset', got %q", result)
		}
	})

	t.Run("Verbose mode multiple intervals", func(t *testing.T) {
		config := OutputConfig{
			Mode:          ModeVerbose,
			MinutesList:   []int{25, 5},
			SecondsList:   []int{0, 0},
			IntervalCount: 2,
		}
		result := FormatResetOutput(config, 0, timestamp)
		if !containsString(result, "interval 1/2") {
			t.Errorf("expected 'interval 1/2', got %q", result)
		}
	})

	t.Run("Non-verbose modes return empty", func(t *testing.T) {
		modes := []OutputMode{ModeDefault, ModeJSON, ModeWatch}
		for _, mode := range modes {
			config := OutputConfig{
				Mode:          mode,
				MinutesList:   []int{25},
				SecondsList:   []int{0},
				IntervalCount: 1,
			}
			result := FormatResetOutput(config, 0, timestamp)
			if result != "" {
				t.Errorf("mode %v: expected empty string, got %q", mode, result)
			}
		}
	})
}

// TestOutputModeConstants tests that output mode constants have correct values
func TestOutputModeConstants(t *testing.T) {
	if ModeDefault != 0 {
		t.Errorf("ModeDefault = %d, want 0", ModeDefault)
	}
	if ModeVerbose != 1 {
		t.Errorf("ModeVerbose = %d, want 1", ModeVerbose)
	}
	if ModeJSON != 2 {
		t.Errorf("ModeJSON = %d, want 2", ModeJSON)
	}
	if ModeWatch != 3 {
		t.Errorf("ModeWatch = %d, want 3", ModeWatch)
	}
}

// TestTimerStateTogglePauseNegativeRemaining tests TogglePause when time has passed
func TestTimerStateTogglePauseNegativeRemaining(t *testing.T) {
	intervals := []time.Duration{1 * time.Millisecond}
	ts := NewTimerState(intervals, []int{0}, []int{0}, false)
	ts.NextBeep = time.Now().Add(-1 * time.Second) // Simulate time passed

	// Pause when time is already expired
	ts.TogglePause()

	// PausedAt should be clamped to 0
	if ts.PausedAt != 0 {
		t.Errorf("PausedAt = %v, want 0 when time has expired", ts.PausedAt)
	}
}

// TestFormatTickOutputVerboseMultipleIntervals tests verbose mode with multiple intervals
func TestFormatTickOutputVerboseMultipleIntervals(t *testing.T) {
	config := OutputConfig{
		Mode:          ModeVerbose,
		MinutesList:   []int{25, 5},
		SecondsList:   []int{0, 0},
		IntervalCount: 2,
	}
	result := FormatTickOutput(config, 4*time.Minute+30*time.Second, 1)
	if !containsString(result, "interval 2/2") {
		t.Errorf("expected 'interval 2/2', got %q", result)
	}
	if !containsString(result, "5m 0s") {
		t.Errorf("expected '5m 0s', got %q", result)
	}
}

// TestVersion tests the version variable
func TestVersion(t *testing.T) {
	t.Run("version has default value", func(t *testing.T) {
		// In tests, version should be "dev" unless ldflags are used
		if version != "dev" && version == "" {
			t.Errorf("expected version to have a value, got empty string")
		}
	})

	t.Run("version can be set", func(t *testing.T) {
		// Save original version
		originalVersion := version
		defer func() {
			version = originalVersion
		}()

		// Set a custom version
		version = "1.2.3-test"
		if version != "1.2.3-test" {
			t.Errorf("expected version to be '1.2.3-test', got %q", version)
		}
	})
}

// containsString is a helper to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
