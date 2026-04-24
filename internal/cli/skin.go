package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Skin defines color and style settings for the CLI.
type Skin struct {
	Name        string     `yaml:"name" json:"name"`
	Description string     `yaml:"description" json:"description"`
	Colors      SkinColors `yaml:"colors" json:"colors"`
}

// SkinColors defines individual color values (ANSI color names or hex).
type SkinColors struct {
	Prompt   string `yaml:"prompt" json:"prompt"`
	Response string `yaml:"response" json:"response"`
	Error    string `yaml:"error" json:"error"`
	Info     string `yaml:"info" json:"info"`
	Code     string `yaml:"code" json:"code"`
	Muted    string `yaml:"muted" json:"muted"`
	Accent   string `yaml:"accent" json:"accent"`
	Banner   string `yaml:"banner" json:"banner"`
}

// SkinEngine manages skins and provides the current active skin.
type SkinEngine struct {
	skins   map[string]*Skin
	current string
}

// NewSkinEngine creates a skin engine with bundled skins loaded.
func NewSkinEngine() *SkinEngine {
	engine := &SkinEngine{
		skins:   make(map[string]*Skin),
		current: "default",
	}
	engine.loadBundled()
	return engine
}

// Current returns the currently active skin.
func (e *SkinEngine) Current() *Skin {
	if s, ok := e.skins[e.current]; ok {
		return s
	}
	return e.skins["default"]
}

// Set changes the active skin. Returns error if skin not found.
func (e *SkinEngine) Set(name string) error {
	if _, ok := e.skins[name]; !ok {
		return fmt.Errorf("skin %q not found", name)
	}
	e.current = name
	return nil
}

// List returns the names of all available skins.
func (e *SkinEngine) List() []string {
	names := make([]string, 0, len(e.skins))
	for name := range e.skins {
		names = append(names, name)
	}
	return names
}

// LoadFromFile loads a skin from a YAML file and adds it to the engine.
func (e *SkinEngine) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read skin file: %w", err)
	}

	var skin Skin
	if err := yaml.Unmarshal(data, &skin); err != nil {
		return fmt.Errorf("parse skin file: %w", err)
	}

	if skin.Name == "" {
		skin.Name = filepath.Base(path)
	}

	e.skins[skin.Name] = &skin
	return nil
}

// LoadFromDir loads all YAML skin files from a directory.
func (e *SkinEngine) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory not existing is not an error
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yaml" || ext == ".yml" {
			if err := e.LoadFromFile(filepath.Join(dir, entry.Name())); err != nil {
				continue // skip invalid skin files
			}
		}
	}
	return nil
}

// Render applies the current skin colors to the given text for the specified element.
// Element names: "prompt", "response", "error", "info", "code", "muted", "accent", "banner".
// Returns the text wrapped in ANSI escape codes for the corresponding color.
func (e *SkinEngine) Render(element, text string) string {
	skin := e.Current()
	if skin == nil {
		return text
	}
	var color string
	switch element {
	case "prompt":
		color = skin.Colors.Prompt
	case "response":
		color = skin.Colors.Response
	case "error":
		color = skin.Colors.Error
	case "info":
		color = skin.Colors.Info
	case "code":
		color = skin.Colors.Code
	case "muted":
		color = skin.Colors.Muted
	case "accent":
		color = skin.Colors.Accent
	case "banner":
		color = skin.Colors.Banner
	default:
		return text
	}
	if color == "" {
		return text
	}
	ansi := colorToANSI(color)
	if ansi == "" {
		return text
	}
	return ansi + text + "\033[0m"
}

// colorToANSI converts a color name or hex code to an ANSI escape sequence.
func colorToANSI(color string) string {
	// Named ANSI colors.
	namedColors := map[string]string{
		"black":   "\033[30m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"gray":    "\033[90m",
		"grey":    "\033[90m",
	}
	if ansi, ok := namedColors[color]; ok {
		return ansi
	}

	// Hex color (#RRGGBB) -> 24-bit ANSI escape.
	if len(color) == 7 && color[0] == '#' {
		r, err1 := strconv.ParseUint(color[1:3], 16, 8)
		g, err2 := strconv.ParseUint(color[3:5], 16, 8)
		b, err3 := strconv.ParseUint(color[5:7], 16, 8)
		if err1 == nil && err2 == nil && err3 == nil {
			return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
		}
	}

	return ""
}

func (e *SkinEngine) loadBundled() {
	e.skins["default"] = &Skin{
		Name:        "default",
		Description: "Clean default theme",
		Colors: SkinColors{
			Prompt:   "cyan",
			Response: "white",
			Error:    "red",
			Info:     "blue",
			Code:     "green",
			Muted:    "gray",
			Accent:   "magenta",
			Banner:   "cyan",
		},
	}

	e.skins["midnight"] = &Skin{
		Name:        "midnight",
		Description: "Dark blue theme for night owls",
		Colors: SkinColors{
			Prompt:   "#5B8DEF",
			Response: "#C0C0C0",
			Error:    "#FF6B6B",
			Info:     "#6BC5FF",
			Code:     "#A8E6CF",
			Muted:    "#666666",
			Accent:   "#DDA0DD",
			Banner:   "#5B8DEF",
		},
	}

	e.skins["solar"] = &Skin{
		Name:        "solar",
		Description: "Warm solar-inspired theme",
		Colors: SkinColors{
			Prompt:   "#FFA500",
			Response: "#FFEFD5",
			Error:    "#FF4500",
			Info:     "#FFD700",
			Code:     "#F0E68C",
			Muted:    "#BDB76B",
			Accent:   "#FF8C00",
			Banner:   "#FFA500",
		},
	}

	e.skins["matrix"] = &Skin{
		Name:        "matrix",
		Description: "Green-on-black hacker theme",
		Colors: SkinColors{
			Prompt:   "#00FF00",
			Response: "#00CC00",
			Error:    "#FF0000",
			Info:     "#00FF00",
			Code:     "#33FF33",
			Muted:    "#006600",
			Accent:   "#00FF00",
			Banner:   "#00FF00",
		},
	}

	e.skins["rose"] = &Skin{
		Name:        "rose",
		Description: "Soft rose-gold theme",
		Colors: SkinColors{
			Prompt:   "#E8A0BF",
			Response: "#F5E6CC",
			Error:    "#FF6B6B",
			Info:     "#B4C7E7",
			Code:     "#C1E1C1",
			Muted:    "#C4A882",
			Accent:   "#E8A0BF",
			Banner:   "#E8A0BF",
		},
	}
}
