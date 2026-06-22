package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Level constants
const (
	LevelInfo    = "info"
	LevelSuccess = "success"
	LevelWarning = "warning"
	LevelError   = "error"
)

// Source constants
const (
	SrcSystem  = "system"
	SrcHID     = "hid"
	SrcWifi    = "wifi"
	SrcTrigger = "trigger"
	SrcExfil   = "exfil"
	SrcGadget  = "gadget"
	SrcNetwork = "network"
)

type Entry struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}

var (
	logFile     string
	fileMu      sync.Mutex
	subscribers []chan Entry
	subMu       sync.Mutex
)

func Init(path string) {
	logFile = path
}

// Core logging function
func Log(level, source, message string) Entry {
	entry := Entry{
		ID:        fmt.Sprintf("%d", time.Now().UnixMilli()),
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level,
		Source:    source,
		Message:   message,
	}

	// Write to file
	fileMu.Lock()
	logs := read()
	logs = append([]Entry{entry}, logs...)
	if len(logs) > 500 {
		logs = logs[:500]
	}
	write(logs)
	fileMu.Unlock()

	// Broadcast to SSE subscribers
	subMu.Lock()
	for _, ch := range subscribers {
		select {
		case ch <- entry:
		default:
		}
	}
	subMu.Unlock()

	return entry
}

// Convenience functions
func Info(source, message string)    { Log(LevelInfo, source, message) }
func Success(source, message string) { Log(LevelSuccess, source, message) }
func Warn(source, message string)    { Log(LevelWarning, source, message) }
func Error(source, message string)   { Log(LevelError, source, message) }

// Get logs with optional filters
func Get(limit int, level, source string) []Entry {
	fileMu.Lock()
	all := read()
	fileMu.Unlock()

	result := make([]Entry, 0)
	for _, e := range all {
		if level != "" && e.Level != level {
			continue
		}
		if source != "" && e.Source != source {
			continue
		}
		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func Clear() {
	fileMu.Lock()
	write([]Entry{})
	fileMu.Unlock()
}

// SSE subscription
func Subscribe() chan Entry {
	ch := make(chan Entry, 64)
	subMu.Lock()
	subscribers = append(subscribers, ch)
	subMu.Unlock()
	return ch
}

func Unsubscribe(ch chan Entry) {
	subMu.Lock()
	defer subMu.Unlock()
	for i, s := range subscribers {
		if s == ch {
			subscribers = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

func read() []Entry {
	if logFile == "" {
		return []Entry{}
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		return []Entry{}
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []Entry{}
	}
	return entries
}

func write(entries []Entry) {
	if logFile == "" {
		return
	}
	data, _ := json.MarshalIndent(entries, "", "  ")
	tmp := logFile + ".tmp"
	_ = os.WriteFile(tmp, data, 0644)
	_ = os.Rename(tmp, logFile)
}
