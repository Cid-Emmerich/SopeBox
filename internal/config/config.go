// Package config handles SopeBox's configuration file and standard paths.
//
// Layout:
//
//	~/.config/sopebox/sopeboxrc          key = value settings
//	~/.local/share/sopebox/library.json  subscriptions, episodes, positions
//	~/.cache/sopebox/icons               podcast artwork
//	~/.cache/sopebox/transcripts         parsed / generated transcripts
//	~/Podcasts/SopeBox                   downloaded episodes
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Config holds every user-tunable setting. Anything not present in the file
// keeps its default, so old config files keep working after upgrades.
type Config struct {
	DownloadDir string

	// Look & feel
	Theme       string // theme name or "match" (follows the podcast icon)
	IconMode    string // "blocks", "ascii", "kitty"
	ShowIcons   bool   // icons in the podcast list
	Captions    bool   // captions pane on / off
	CaptionSide string // "right", "bottom"
	CaptionSize int    // percent of the screen given to captions
	Emoji       bool   // animated emoji in the captions pane
	EmojiStyle  string // "blocks", "braille", "ascii", "chunky"
	EmojiColour string // "emoji", "theme", "rainbow", ...
	Names       bool   // speaker names under the visualizer

	// Playback
	Volume   float64 // 0..1
	Speed    float64 // 0.5..3
	SkipFwd  int     // seconds for →
	SkipBack int     // seconds for ←
	Continue bool    // keep playing the next episode in the queue

	// Visualizer
	Vis          string
	VisGradient  string
	VisFill      string
	VisPeaks     bool
	VisMirror    bool
	VisSmoothing float64
	VisGain      float64
	VisFalloff   float64
	VisRotate    float64 // orb rotation speed
	VisOrbSize   float64 // 0.5..2
	VisPhysics   bool    // orbs drift and repel
	VisTrails    bool
	VisFPS       int

	// Voice isolation
	VoiceMode        string  // "auto", "transcript", "acoustic"
	VoiceSensitivity float64 // 0..1 how eager to spawn a new orb
	VoiceSmoothing   float64 // 0..0.95 feature smoothing
	VoiceFalloff     float64 // how fast an idle orb dims
	VoiceMax         int     // max orbs on screen

	// Downloads
	AutoDownload int  // 0, 1, 3, 5, 10 most recent per podcast
	Concurrency  int  // parallel downloads
	KeepLatest   int  // delete downloads older than the N latest (0 = keep all)
	RefreshStart bool // refresh feeds when the UI opens

	// Transcription
	WhisperBin     string // path to whisper-cli (auto-detected if empty)
	WhisperModel   string // "tiny.en", "base.en", "small.en", ...
	AutoTranscribe string // "off", "downloaded", "always"

	// Collage (pictures of who and what is mentioned)
	CollageModel  string // Claude model that reads transcripts
	CollageImages string // "kitty" (real pictures) or "blocks"
	AnthropicKey  string // API key; ANTHROPIC_API_KEY in the environment wins

	// Paths
	LibraryPath string
	CacheDir    string
	ConfigPath  string
}

// Default returns the baseline configuration.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DownloadDir:      filepath.Join(home, "Podcasts", "SopeBox"),
		Theme:            "sopebox",
		IconMode:         "blocks",
		ShowIcons:        true,
		Captions:         true,
		CaptionSide:      "right",
		CaptionSize:      38,
		Emoji:            true,
		EmojiStyle:       "blocks",
		EmojiColour:      "emoji",
		Names:            true,
		Volume:           0.8,
		Speed:            1.0,
		SkipFwd:          30,
		SkipBack:         10,
		Continue:         true,
		Vis:              "orbs",
		VisGradient:      "theme",
		VisFill:          "braille",
		VisPeaks:         true,
		VisMirror:        false,
		VisSmoothing:     0.55,
		VisGain:          1.0,
		VisFalloff:       0.08,
		VisRotate:        0.15,
		VisOrbSize:       1.0,
		VisPhysics:       true,
		VisTrails:        false,
		VisFPS:           30,
		VoiceMode:        "auto",
		VoiceSensitivity: 0.5,
		VoiceSmoothing:   0.7,
		VoiceFalloff:     0.06,
		VoiceMax:         6,
		AutoDownload:     0,
		Concurrency:      2,
		KeepLatest:       0,
		RefreshStart:     true,
		WhisperBin:       "",
		WhisperModel:     "base.en",
		AutoTranscribe:   "off",
		CollageModel:     "claude-opus-5-5",
		CollageImages:    "kitty",
		LibraryPath:      filepath.Join(dataDir(), "library.json"),
		CacheDir:         cacheDir(),
		ConfigPath:       filepath.Join(configDir(), "sopeboxrc"),
	}
}

func configDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "sopebox")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "sopebox")
}

func cacheDir() string {
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "sopebox")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "sopebox")
}

func dataDir() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "sopebox")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "sopebox")
}

// Load reads the config file. A missing file simply yields the defaults.
func Load() (Config, error) {
	c := Default()
	f, err := os.Open(c.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		c.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	c.DownloadDir = ExpandHome(c.DownloadDir)
	return c, sc.Err()
}

// ExpandHome turns a leading "~" into the user's home directory.
func ExpandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}

// Set applies one key = value pair. Unknown keys are ignored.
func (c *Config) Set(k, v string) {
	b := func() bool { return v == "true" || v == "1" || v == "yes" || v == "on" }
	i := func() int { n, _ := strconv.Atoi(v); return n }
	f := func() float64 { n, _ := strconv.ParseFloat(v, 64); return n }
	switch k {
	case "download_dir", "path":
		c.DownloadDir = v
	case "theme":
		c.Theme = v
	case "icon_mode":
		c.IconMode = v
	case "show_icons":
		c.ShowIcons = b()
	case "captions":
		c.Captions = b()
	case "caption_side":
		c.CaptionSide = v
	case "caption_size":
		c.CaptionSize = int(Clamp(float64(i()), 20, 70))
	case "emoji":
		c.Emoji = b()
	case "emoji_style":
		c.EmojiStyle = v
	case "emoji_colour", "emoji_color":
		c.EmojiColour = v
	case "names":
		c.Names = b()
	case "volume":
		c.Volume = Clamp(f(), 0, 1)
	case "speed":
		c.Speed = Clamp(f(), 0.5, 3)
	case "skip_forward":
		c.SkipFwd = i()
	case "skip_back":
		c.SkipBack = i()
	case "continue":
		c.Continue = b()
	case "vis":
		c.Vis = v
	case "vis_gradient":
		c.VisGradient = v
	case "vis_fill":
		c.VisFill = v
	case "vis_peaks":
		c.VisPeaks = b()
	case "vis_mirror":
		c.VisMirror = b()
	case "vis_smoothing":
		c.VisSmoothing = Clamp(f(), 0, 0.95)
	case "vis_gain":
		c.VisGain = Clamp(f(), 0.1, 10)
	case "vis_falloff":
		c.VisFalloff = Clamp(f(), 0.01, 1)
	case "vis_rotate":
		c.VisRotate = Clamp(f(), -2, 2)
	case "vis_orb_size":
		c.VisOrbSize = Clamp(f(), 0.4, 2.5)
	case "vis_physics":
		c.VisPhysics = b()
	case "vis_trails":
		c.VisTrails = b()
	case "vis_fps":
		c.VisFPS = i()
	case "voice_mode":
		c.VoiceMode = v
	case "voice_sensitivity":
		c.VoiceSensitivity = Clamp(f(), 0, 1)
	case "voice_smoothing":
		c.VoiceSmoothing = Clamp(f(), 0, 0.95)
	case "voice_falloff":
		c.VoiceFalloff = Clamp(f(), 0.005, 0.5)
	case "voice_max":
		c.VoiceMax = int(Clamp(float64(i()), 1, 12))
	case "auto_download":
		c.AutoDownload = i()
	case "concurrency":
		c.Concurrency = int(Clamp(float64(i()), 1, 8))
	case "keep_latest":
		c.KeepLatest = i()
	case "refresh_on_start":
		c.RefreshStart = b()
	case "whisper_bin":
		c.WhisperBin = v
	case "whisper_model":
		c.WhisperModel = v
	case "auto_transcribe":
		c.AutoTranscribe = v
	case "collage_model":
		c.CollageModel = v
	case "collage_images":
		c.CollageImages = v
	case "anthropic_api_key":
		c.AnthropicKey = v
	}
}

// Clamp limits x to [lo, hi].
func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// Save regenerates the config file with every current value.
func (c Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.ConfigPath), 0o755); err != nil {
		return err
	}
	kv := map[string]string{
		"download_dir":      c.DownloadDir,
		"theme":             c.Theme,
		"icon_mode":         c.IconMode,
		"show_icons":        fmt.Sprint(c.ShowIcons),
		"captions":          fmt.Sprint(c.Captions),
		"caption_side":      c.CaptionSide,
		"caption_size":      fmt.Sprint(c.CaptionSize),
		"emoji":             fmt.Sprint(c.Emoji),
		"emoji_style":       c.EmojiStyle,
		"emoji_colour":      c.EmojiColour,
		"names":             fmt.Sprint(c.Names),
		"volume":            fmt.Sprintf("%.2f", c.Volume),
		"speed":             fmt.Sprintf("%.2f", c.Speed),
		"skip_forward":      fmt.Sprint(c.SkipFwd),
		"skip_back":         fmt.Sprint(c.SkipBack),
		"continue":          fmt.Sprint(c.Continue),
		"vis":               c.Vis,
		"vis_gradient":      c.VisGradient,
		"vis_fill":          c.VisFill,
		"vis_peaks":         fmt.Sprint(c.VisPeaks),
		"vis_mirror":        fmt.Sprint(c.VisMirror),
		"vis_smoothing":     fmt.Sprintf("%.2f", c.VisSmoothing),
		"vis_gain":          fmt.Sprintf("%.2f", c.VisGain),
		"vis_falloff":       fmt.Sprintf("%.2f", c.VisFalloff),
		"vis_rotate":        fmt.Sprintf("%.2f", c.VisRotate),
		"vis_orb_size":      fmt.Sprintf("%.2f", c.VisOrbSize),
		"vis_physics":       fmt.Sprint(c.VisPhysics),
		"vis_trails":        fmt.Sprint(c.VisTrails),
		"vis_fps":           fmt.Sprint(c.VisFPS),
		"voice_mode":        c.VoiceMode,
		"voice_sensitivity": fmt.Sprintf("%.2f", c.VoiceSensitivity),
		"voice_smoothing":   fmt.Sprintf("%.2f", c.VoiceSmoothing),
		"voice_falloff":     fmt.Sprintf("%.3f", c.VoiceFalloff),
		"voice_max":         fmt.Sprint(c.VoiceMax),
		"auto_download":     fmt.Sprint(c.AutoDownload),
		"concurrency":       fmt.Sprint(c.Concurrency),
		"keep_latest":       fmt.Sprint(c.KeepLatest),
		"refresh_on_start":  fmt.Sprint(c.RefreshStart),
		"whisper_bin":       c.WhisperBin,
		"whisper_model":     c.WhisperModel,
		"auto_transcribe":   c.AutoTranscribe,
		"collage_model":     c.CollageModel,
		"collage_images":    c.CollageImages,
	}
	if c.AnthropicKey != "" {
		kv["anthropic_api_key"] = c.AnthropicKey
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString("# SopeBox configuration\n")
	sb.WriteString("# Edit by hand, or change settings inside the player (saved on quit).\n")
	sb.WriteString("# Run `sopebox path <dir>` to change the download folder.\n\n")
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s = %s\n", k, kv[k])
	}
	// owner-only: the file may hold an API key (WriteFile keeps the mode
	// of an existing file, so set it explicitly)
	if err := os.WriteFile(c.ConfigPath, []byte(sb.String()), 0o600); err != nil {
		return err
	}
	return os.Chmod(c.ConfigPath, 0o600)
}
