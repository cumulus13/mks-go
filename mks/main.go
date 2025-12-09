package main

import (
	// "bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
)

type ParseResult struct {
	indent int
	name   string
	isDir  bool
}

func parseTreeLine(line string) (*ParseResult, error) {
	line = strings.TrimRight(line, " \t\r\n")
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	// Remove comments
	if idx := strings.Index(line, "#"); idx != -1 {
		line = strings.TrimRight(line[:idx], " \t")
	}

	if line == "" {
		return nil, fmt.Errorf("empty after comment")
	}

	// Extract name after tree pattern
	var namePart string
	if idx := strings.Index(line, "├── "); idx != -1 {
		namePart = line[idx+len("├── "):]
	} else if idx := strings.Index(line, "└── "); idx != -1 {
		namePart = line[idx+len("└── "):]
	} else {
		// Fallback for root or other formats
		parts := strings.Fields(line)
		if len(parts) > 0 {
			namePart = parts[len(parts)-1]
		} else {
			namePart = line
		}
	}

	namePart = strings.TrimSpace(namePart)
	if namePart == "" {
		return nil, fmt.Errorf("no name found")
	}

	isDir := strings.HasSuffix(namePart, "/")
	name := strings.TrimSpace(strings.TrimSuffix(namePart, "/"))

	if name == "" || !isValidFilename(name) {
		return nil, fmt.Errorf("invalid file name")
	}

	// Calculate indent by counting characters before name
	charsBeforeName := 0
	for _, ch := range line {
		if strings.HasPrefix(namePart, string(ch)) {
			break
		}
		charsBeforeName++
	}

	indent := charsBeforeName / 4

	return &ParseResult{
		indent: indent,
		name:   name,
		isDir:  isDir,
	}, nil
}

func isValidFilename(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}

	// Check reserved names (Windows)
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

	// Check illegal characters
	illegal := `<>:"/\|?*`
	for _, ch := range illegal {
		if strings.ContainsRune(name, ch) {
			return false
		}
	}

	// Cannot end with space or dot (Windows)
	if strings.HasSuffix(trimmed, " ") || strings.HasSuffix(trimmed, ".") {
		return false
	}

	return true
}

func looksLikeTree(content string) bool {
	treeMarkers := []string{"├", "└", "─", "│", "┬", "┼"}
	for _, marker := range treeMarkers {
		if strings.Contains(content, marker) {
			return strings.Count(content, "\n") >= 1
		}
	}

	// Try indentation detection
	lines := strings.Split(content, "\n")
	indentedLines := 0
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed != "" && len(line) > len(trimmed) {
			indentedLines++
		}
	}

	return indentedLines >= 2 && len(lines) >= 2
}

func createStructure(lines []string, debug bool) error {
	var pathStack []string

	for idx, line := range lines {
		parsed, err := parseTreeLine(line)
		if err != nil {
			continue
		}

		indent := parsed.indent
		name := parsed.name
		isDir := parsed.isDir

		if debug {
			fmt.Printf("[DEBUG] Line %d: indent=%d, name='%s', isDir=%v\n", idx, indent, name, isDir)
			fmt.Printf("[DEBUG] Stack before: %v\n", pathStack)
		}

		if len(pathStack) == 0 {
			// Root
			if isDir {
				if err := os.MkdirAll(name, 0755); err != nil {
					return err
				}
				pathStack = append(pathStack, name)
				if debug {
					fmt.Printf("📁 Root: %s\n", name)
				}
			} else {
				if err := os.WriteFile(name, []byte{}, 0644); err != nil {
					return err
				}
				if debug {
					fmt.Printf("📄 Root file: %s\n", name)
				}
			}
			continue
		}

		// Adjust stack based on indent
		if indent > len(pathStack) {
			if debug {
				fmt.Printf("⚠️ Warning: indent %d > stack size %d\n", indent, len(pathStack))
			}
		} else {
			pathStack = pathStack[:indent]
		}

		if debug {
			fmt.Printf("[DEBUG] Stack after truncate: %v\n", pathStack)
		}

		// Build full path
		fullPath := filepath.Join(append(pathStack, name)...)

		if isDir {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return err
			}
			pathStack = append(pathStack, name)
			if debug {
				fmt.Printf("📁 %s\n", fullPath)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fullPath, []byte{}, 0644); err != nil {
				return err
			}
			if debug {
				fmt.Printf("📄 %s\n", fullPath)
			}
		}

		if debug {
			fmt.Printf("[DEBUG] Stack after: %v\n\n", pathStack)
		}
	}

	return nil
}

func readInput(args []string) ([]string, string, error) {
	debug := false
	var filePath string

	for i, arg := range args {
		if arg == "--debug" {
			debug = true
		} else if i > 0 && !debug {
			filePath = arg
		} else if i > 0 && debug && args[i-1] != "--debug" {
			filePath = arg
		}
	}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", err
		}
		lines := strings.Split(string(data), "\n")
		return lines, "file", nil
	}

	content, err := clipboard.ReadAll()
	if err != nil {
		return nil, "", fmt.Errorf("clipboard read failed: %v", err)
	}

	if strings.TrimSpace(content) == "" {
		return nil, "", fmt.Errorf("clipboard is empty")
	}

	if !looksLikeTree(content) {
		return nil, "", fmt.Errorf("clipboard is not a tree-structure")
	}

	lines := strings.Split(content, "\n")
	return lines, "clipboard", nil
}

func isValidStructure(lines []string) bool {
	for _, line := range lines {
		if _, err := parseTreeLine(line); err == nil {
			return true
		}
	}
	return false
}

func main() {
	debug := false
	for _, arg := range os.Args {
		if arg == "--debug" {
			debug = true
			break
		}
	}

	lines, source, err := readInput(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if !isValidStructure(lines) {
		fmt.Fprintln(os.Stderr, "❌ Input is empty or invalid.")
		os.Exit(1)
	}

	fmt.Printf("📋 Read from %s (%d lines)\n", source, len(lines))

	if debug {
		fmt.Println("🐛 Debug mode enabled\n")
	}

	fmt.Println("✅ Creating structure...\n")

	if err := createStructure(lines, debug); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Done!")
}