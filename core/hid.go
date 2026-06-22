package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const HIDDevice = "/dev/hidg0"

var hidMu sync.Mutex

// HID keymaps
var keymap = map[rune][2]byte{
	'a': {0, 4}, 'b': {0, 5}, 'c': {0, 6}, 'd': {0, 7},
	'e': {0, 8}, 'f': {0, 9}, 'g': {0, 10}, 'h': {0, 11},
	'i': {0, 12}, 'j': {0, 13}, 'k': {0, 14}, 'l': {0, 15},
	'm': {0, 16}, 'n': {0, 17}, 'o': {0, 18}, 'p': {0, 19},
	'q': {0, 20}, 'r': {0, 21}, 's': {0, 22}, 't': {0, 23},
	'u': {0, 24}, 'v': {0, 25}, 'w': {0, 26}, 'x': {0, 27},
	'y': {0, 28}, 'z': {0, 29},
	'A': {2, 4}, 'B': {2, 5}, 'C': {2, 6}, 'D': {2, 7},
	'E': {2, 8}, 'F': {2, 9}, 'G': {2, 10}, 'H': {2, 11},
	'I': {2, 12}, 'J': {2, 13}, 'K': {2, 14}, 'L': {2, 15},
	'M': {2, 16}, 'N': {2, 17}, 'O': {2, 18}, 'P': {2, 19},
	'Q': {2, 20}, 'R': {2, 21}, 'S': {2, 22}, 'T': {2, 23},
	'U': {2, 24}, 'V': {2, 25}, 'W': {2, 26}, 'X': {2, 27},
	'Y': {2, 28}, 'Z': {2, 29},
	'1': {0, 30}, '2': {0, 31}, '3': {0, 32}, '4': {0, 33},
	'5': {0, 34}, '6': {0, 35}, '7': {0, 36}, '8': {0, 37},
	'9': {0, 38}, '0': {0, 39},
	'!': {2, 30}, '@': {2, 31}, '#': {2, 32}, '$': {2, 33},
	'%': {2, 34}, '^': {2, 35}, '&': {2, 36}, '*': {2, 37},
	'(': {2, 38}, ')': {2, 39},
	' ': {0, 44}, '\n': {0, 40}, '\t': {0, 43},
	'-': {0, 45}, '_': {2, 45}, '=': {0, 46}, '+': {2, 46},
	'[': {0, 47}, '{': {2, 47}, ']': {0, 48}, '}': {2, 48},
	'\\': {0, 49}, '|': {2, 49}, ';': {0, 51}, ':': {2, 51},
	'\'': {0, 52}, '"': {2, 52}, '`': {0, 53}, '~': {2, 53},
	',': {0, 54}, '<': {2, 54}, '.': {0, 55}, '>': {2, 55},
	'/': {0, 56}, '?': {2, 56},
}

var specialKeys = map[string][2]byte{
	"ENTER":     {0, 40}, "TAB":      {0, 43}, "SPACE":    {0, 44},
	"BACKSPACE": {0, 42}, "ESC":      {0, 41}, "DELETE":   {0, 76},
	"UP":        {0, 82}, "DOWN":     {0, 81}, "LEFT":     {0, 80},
	"RIGHT":     {0, 79}, "HOME":     {0, 74}, "END":      {0, 77},
	"PAGEUP":    {0, 75}, "PAGEDOWN": {0, 78},
	"F1":  {0, 58}, "F2":  {0, 59}, "F3":  {0, 60}, "F4":  {0, 61},
	"F5":  {0, 62}, "F6":  {0, 63}, "F7":  {0, 64}, "F8":  {0, 65},
	"F9":  {0, 66}, "F10": {0, 67}, "F11": {0, 68}, "F12": {0, 69},
	"GUI": {8, 0}, "WINDOWS": {8, 0},
	"CTRL": {1, 0}, "ALT": {4, 0}, "SHIFT": {2, 0},
}

func sendKey(mod, keycode byte) error {
	f, err := os.OpenFile(HIDDevice, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", HIDDevice, err)
	}
	defer f.Close()

	// Key down
	report := [8]byte{mod, 0, keycode, 0, 0, 0, 0, 0}
	if _, err := f.Write(report[:]); err != nil {
		return fmt.Errorf("write key down: %w", err)
	}
	time.Sleep(20 * time.Millisecond)

	// Key up
	release := [8]byte{}
	if _, err := f.Write(release[:]); err != nil {
		return fmt.Errorf("write key up: %w", err)
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func sendString(text string) error {
	for _, ch := range text {
		if k, ok := keymap[ch]; ok {
			if err := sendKey(k[0], k[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

// RunDucky executes a Ducky Script
func RunDucky(script string) error {
	hidMu.Lock()
	defer hidMu.Unlock()

	lines := strings.Split(strings.TrimSpace(script), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "REM"):
			continue

		case strings.HasPrefix(line, "DELAY "):
			ms, err := strconv.Atoi(strings.TrimPrefix(line, "DELAY "))
			if err == nil {
				time.Sleep(time.Duration(ms) * time.Millisecond)
			}

		case strings.HasPrefix(line, "STRING "):
			if err := sendString(strings.TrimPrefix(line, "STRING ")); err != nil {
				return err
			}

		default:
			// Special key or combo (e.g. "GUI r", "CTRL ALT DELETE")
			if k, ok := specialKeys[line]; ok {
				if err := sendKey(k[0], k[1]); err != nil {
					return err
				}
				continue
			}
			// Key combo
			parts := strings.Fields(line)
			var mod, keycode byte
			for _, part := range parts {
				if k, ok := specialKeys[part]; ok {
					mod |= k[0]
					if k[1] != 0 {
						keycode = k[1]
					}
				} else if len(part) == 1 {
					if k, ok := keymap[rune(part[0])]; ok {
						mod |= k[0]
						if k[1] != 0 {
							keycode = k[1]
						}
					}
				}
			}
			if mod != 0 || keycode != 0 {
				if err := sendKey(mod, keycode); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// HIDAvailable checks if the HID device exists
func HIDAvailable() bool {
	_, err := os.Stat(HIDDevice)
	return err == nil
}
