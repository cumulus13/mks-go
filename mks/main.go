package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
)

// parseTreeLine mengembalikan (indentLevel, name, isDir)
func parseTreeLine(line string) (int, string, bool, error) {
	// Hapus komentar
	if i := strings.Index(line, "#"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimRight(line, " \t\r\n")
	if line == "" {
		return 0, "", false, fmt.Errorf("empty")
	}

	// Ambil seluruh bagian setelah karakter terakhir yang bukan bagian dari nama
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, "", false, fmt.Errorf("no fields")
	}

	rawName := fields[len(fields)-1]
	isDir := strings.HasSuffix(rawName, "/")
	name := strings.TrimSuffix(rawName, "/")
	name = strings.TrimSpace(name)

	if name == "" || !isValidFileName(name) {
		return 0, "", false, fmt.Errorf("invalid name: %q", name)
	}

	// Sekarang hitung indent: cari posisi awal nama asli dalam line
	nameStart := strings.LastIndex(line, name)
	if nameStart == -1 {
		// fallback: anggap indent 0
		return 0, name, isDir, nil
	}

	prefix := line[:nameStart]
	// Normalisasi prefix: ganti semua non-spasi jadi spasi
	var norm strings.Builder
	for _, c := range prefix {
		if c == ' ' || c == '\t' {
			norm.WriteRune(c)
		} else {
			norm.WriteRune(' ')
		}
	}
	spaceCount := 0
	for _, c := range norm.String() {
		if c == ' ' {
			spaceCount++
		} else {
			break
		}
	}
	indent := spaceCount / 4

	return indent, name, isDir, nil
}
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

func createStructure(lines []string) error {
	var pathStack []string

	for _, line := range lines {
		indent, name, isDir, err := parseTreeLine(line)
		if err != nil {
			continue // skip baris kosong/komentar
		}

		// Handle root (indent 0 dan stack kosong)
		if len(pathStack) == 0 {
			if !isValidFileName(name) {
				return fmt.Errorf("invalid root name: %q", name)
			}
			if err := os.MkdirAll(name, 0755); err != nil {
				return err
			}
			if isDir {
				pathStack = []string{name}
			} else {
				// Root adalah file — jarang, tapi mungkin
				f, err := os.Create(name)
				if err != nil {
					return err
				}
				f.Close()
				// Tidak push ke stack karena bukan direktori
			}
			continue
		}

		// Sesuaikan stack sesuai indent
		if indent < 0 {
			indent = 0
		}
		if indent >= len(pathStack) {
			// Tidak boleh lompat level, batasi ke level terakhir
			indent = len(pathStack) - 1
		}
		pathStack = pathStack[:indent+1]

		fullPath := filepath.Join(append(pathStack, name)...)

		if isDir {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %v", fullPath, err)
			}
			pathStack = append(pathStack, name)
		} else {
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create parent dir %s: %v", dir, err)
			}
			f, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", fullPath, err)
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

func isValidStructure(lines []string) bool {
	for _, line := range lines {
		_, _, _, err := parseTreeLine(line)
		if err == nil {
			return true
		}
	}
	return false
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