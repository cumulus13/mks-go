package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
)

func isValidFileName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	upper := strings.ToUpper(trimmed)
	base := strings.Split(upper, ".")[0]
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	for _, r := range reserved {
		if base == r {
			return false
		}
	}
	for _, c := range `<>:"/\|?*` {
		if strings.ContainsRune(name, c) {
			return false
		}
	}
	if strings.HasSuffix(trimmed, " ") || strings.HasSuffix(trimmed, ".") {
		return false
	}
	return true
}

func extractNameFromLine(line string) (name string, isDir bool) {
	trimmed := strings.TrimRight(line, "\r\n")
	if strings.HasSuffix(trimmed, "/") {
		isDir = true
		trimmed = strings.TrimSuffix(trimmed, "/")
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", false
	}
	name = fields[len(fields)-1]
	return name, isDir
}

func getIndentLevel(line string) int {
	// Count visual indent: each "│   " = 1 level, each 4 spaces = 1 level
	level := 0
	i := 0
	for i < len(line) {
		if strings.HasPrefix(line[i:], "│   ") {
			level++
			i += 4
		} else if line[i] == ' ' {
			// Count spaces in groups of 4
			spaces := 0
			for i < len(line) && line[i] == ' ' {
				spaces++
				i++
			}
			level += spaces / 4
		} else if line[i] == '\t' {
			level++
			i++
		} else {
			break
		}
	}
	return level
}

func createStructure(lines []string) error {
	var pathStack []string

	for i, line := range lines {
		stripped := strings.TrimRight(strings.TrimLeft(line, " \t"), " \t\r\n")
		if stripped == "" {
			continue
		}

		// First line ending with / is root
		if i == 0 && strings.HasSuffix(stripped, "/") {
			root := strings.TrimSuffix(stripped, "/")
			if !isValidFileName(root) {
				return fmt.Errorf("invalid root name: %q", root)
			}
			if err := os.MkdirAll(root, 0755); err != nil {
				return err
			}
			pathStack = []string{root}
			continue
		}

		name, isDir := extractNameFromLine(line)
		if !isValidFileName(name) {
			return fmt.Errorf("invalid name at line %d: %q", i+1, name)
		}

		level := getIndentLevel(line)

		if len(pathStack) > 0 {
			if level >= len(pathStack) {
				level = len(pathStack) - 1
			}
		} else {
			level = 0
		}

		pathStack = pathStack[:level+1]

		fullPath := filepath.Join(append(pathStack, name)...)

		if isDir {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return err
			}
			pathStack = append(pathStack, name)
		} else {
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			f, err := os.Create(fullPath)
			if err != nil {
				return err
			}
			f.Close()
		}
	}

	return nil
}

func readInput() ([]string, string, error) {
	if len(os.Args) > 1 {
		content, err := os.ReadFile(os.Args[1])
		if err != nil {
			return nil, "", err
		}
		return strings.Split(string(content), "\n"), "file", nil
	}

	content, err := clipboard.ReadAll()
	if err != nil {
		return nil, "", fmt.Errorf("clipboard error: %v", err)
	}
	if content == "" {
		return nil, "", fmt.Errorf("clipboard is empty")
	}
	return strings.Split(content, "\n"), "clipboard", nil
}

func main() {
	lines, source, err := readInput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Input error: %v\n", err)
		os.Exit(1)
	}

	if !isValidStructure(lines) {
		fmt.Fprintln(os.Stderr, "❌ Input is empty or invalid.")
		os.Exit(1)
	}

	fmt.Printf("Read from %s (%d lines)\n", source, len(lines))
	fmt.Println("✅ Creating structure...")

	if err := createStructure(lines); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Done!")
}

func isValidStructure(lines []string) bool {
	count := 0
	for _, line := range lines {
		if strings.TrimRight(strings.TrimLeft(line, " \t"), " \t\r\n") != "" {
			count++
		}
	}
	return count > 0
}