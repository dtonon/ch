package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// ANSI color codes
const (
	Reset = "\033[0m"
	Bell  = "\a"
)

type namedColor struct {
	name    string
	r, g, b int
}

// Preset colors in order of assignment
var namedColors = []namedColor{
	{"red", 255, 105, 97},
	{"green", 134, 194, 29},
	{"orange", 240, 160, 75},
	{"blue", 134, 176, 189},
	{"pink", 255, 164, 164},
	{"purple", 203, 166, 247},
}

func rgbToANSI(r, g, b int, background bool) string {
	if background {
		return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

type wordConfig struct {
	original   string
	search     string // lowercase version for case-insensitive search
	color      string
	background bool
}

func parseColor(colorStr string, background bool) string {
	// Check if it's a named color
	lowerColor := strings.ToLower(colorStr)
	for _, nc := range namedColors {
		if nc.name == lowerColor {
			return rgbToANSI(nc.r, nc.g, nc.b, background)
		}
	}

	// Otherwise treat as hex color
	hex := strings.TrimPrefix(colorStr, "#")

	if len(hex) != 6 {
		return ""
	}

	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return rgbToANSI(r, g, b, background)
}

func parseArgs(args []string, caseSensitive bool, background bool) []wordConfig {
	var configs []wordConfig
	usedColors := make(map[int]bool) // track indices in namedColors

	// First pass: reserve colors that are explicitly specified
	for _, arg := range args {
		parts := strings.Split(arg, "::")
		if len(parts) == 2 && parts[1] != "" {
			color := parseColor(parts[1], background)
			if color != "" {
				// Mark named color as used if it matches one of our presets
				for i, nc := range namedColors {
					if color == rgbToANSI(nc.r, nc.g, nc.b, background) {
						usedColors[i] = true
						break
					}
				}
			}
		}
	}

	// Second pass: assign colors
	for _, arg := range args {
		parts := strings.Split(arg, "::")
		word := parts[0]

		var color string
		if len(parts) == 2 && parts[1] != "" {
			// Custom color specified (either named or hex)
			color = parseColor(parts[1], background)
			if color == "" {
				fmt.Fprintf(os.Stderr, "Warning: invalid color '%s' for word '%s', using preset\n", parts[1], word)
				color = getNextAvailableColor(usedColors, background)
			}
		} else {
			// Use next available preset color
			color = getNextAvailableColor(usedColors, background)
		}

		search := word
		if !caseSensitive {
			search = strings.ToLower(word)
		}

		configs = append(configs, wordConfig{
			original:   word,
			search:     search,
			color:      color,
			background: background,
		})
	}

	return configs
}

func getNextAvailableColor(usedColors map[int]bool, background bool) string {
	// Find first unused color from namedColors slice
	for i, nc := range namedColors {
		if !usedColors[i] {
			usedColors[i] = true
			return rgbToANSI(nc.r, nc.g, nc.b, background)
		}
	}

	// If all colors used, cycle back to the beginning
	return rgbToANSI(namedColors[0].r, namedColors[0].g, namedColors[0].b, background)
}

func highlightLine(line string, configs []wordConfig, caseSensitive, wholeWord bool) (string, bool) {
	if len(configs) == 0 {
		return line, false
	}

	searchLine := line
	if !caseSensitive {
		searchLine = strings.ToLower(line)
	}

	// Track which positions are already colored (to handle overlapping matches)
	colored := make([]bool, len(line))

	// Store replacements as [start, end, replacement]
	type replacement struct {
		start int
		end   int
		text  string
	}
	var replacements []replacement

	// Track if we found any matches
	foundMatch := false

	// Find all matches
	for _, cfg := range configs {
		pos := 0
		for {
			idx := strings.Index(searchLine[pos:], cfg.search)
			if idx == -1 {
				break
			}
			idx += pos

			startIdx := idx
			endIdx := idx + len(cfg.search)

			// If wholeWord mode, extend to word boundaries
			if wholeWord {
				// Extend backwards to start of word
				for startIdx > 0 && line[startIdx-1] != ' ' && line[startIdx-1] != '\n' && line[startIdx-1] != '\t' {
					startIdx--
				}
				// Extend forwards to end of word
				for endIdx < len(line) && line[endIdx] != ' ' && line[endIdx] != '\n' && line[endIdx] != '\t' {
					endIdx++
				}
			}

			// Check if this position is already colored (overlapping match)
			alreadyColored := false
			for i := startIdx; i < endIdx; i++ {
				if colored[i] {
					alreadyColored = true
					break
				}
			}

			if !alreadyColored {
				// Mark as colored
				for i := startIdx; i < endIdx; i++ {
					colored[i] = true
				}

				// Store replacement
				matchedText := line[startIdx:endIdx]
				coloredText := cfg.color + matchedText + Reset
				replacements = append(replacements, replacement{
					start: startIdx,
					end:   endIdx,
					text:  coloredText,
				})

				foundMatch = true
			}

			pos = idx + 1
		}
	}

	// If no matches, return original line
	if len(replacements) == 0 {
		return line, false
	}

	// Sort replacements by start position (they should already be mostly sorted)
	// Build result string
	var result strings.Builder
	lastPos := 0

	// Sort replacements by start position
	for i := 0; i < len(replacements); i++ {
		for j := i + 1; j < len(replacements); j++ {
			if replacements[j].start < replacements[i].start {
				replacements[i], replacements[j] = replacements[j], replacements[i]
			}
		}
	}

	for _, r := range replacements {
		result.WriteString(line[lastPos:r.start])
		result.WriteString(r.text)
		lastPos = r.end
	}
	result.WriteString(line[lastPos:])

	return result.String(), foundMatch
}

func main() {
	caseSensitive := flag.Bool("s", false, "case-sensitive matching")
	wholeWord := flag.Bool("w", false, "extend match to whole word (until space or EOL)")
	background := flag.Bool("b", false, "use background colors instead of foreground")
	alert := flag.Bool("a", false, "beep on match (at most once every 5 seconds)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ch [options] <word1> <word2>::<COLOR> ...\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -s    case-sensitive matching (default: case-insensitive)\n")
		fmt.Fprintf(os.Stderr, "  -w    extend match to whole word\n")
		fmt.Fprintf(os.Stderr, "  -b    use background colors instead of foreground\n")
		fmt.Fprintf(os.Stderr, "  -a    beep on match (at most once every 5 seconds)\n")
		fmt.Fprintf(os.Stderr, "\nColors:\n")
		fmt.Fprintf(os.Stderr, "  Named: red, green, orange, blue, pink, purple\n")
		fmt.Fprintf(os.Stderr, "  Hex: any 6-digit hex color (e.g., FF5500)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  tail -f app.log | ch error::red warning::orange success::green\n")
		fmt.Fprintf(os.Stderr, "  tail -f app.log | ch -a error warning  # beep on errors/warnings\n")
		os.Exit(1)
	}

	configs := parseArgs(args, *caseSensitive, *background)

	// Track last beep time for rate limiting
	var lastBeepTime time.Time
	// Read from stdin line by line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		highlighted, foundMatch := highlightLine(line, configs, *caseSensitive, *wholeWord)

		// Beep if alert flag is set, match was found, and enough time has passed
		if *alert && foundMatch {
			now := time.Now()
			if now.Sub(lastBeepTime) >= 5*time.Second {
				fmt.Print(Bell)
				lastBeepTime = now
			}
		}

		fmt.Println(highlighted)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
