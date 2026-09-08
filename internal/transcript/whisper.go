package transcript

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
)

// Models lists the whisper.cpp models SopeBox knows how to fetch, smallest
// first. English-only models are faster and more accurate for English.
var Models = []string{"tiny.en", "base.en", "small.en", "medium.en", "tiny", "base", "small", "medium", "large-v3-turbo", "small.en-tdrz"}

var (
	whisperOnce sync.Once
	whisperPath string
)

// FindWhisper locates a whisper.cpp binary. An explicit path wins.
func FindWhisper(explicit string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
	}
	whisperOnce.Do(func() {
		for _, name := range []string{"whisper-cli", "whisper-cpp", "whisper.cpp", "whisper-main"} {
			if p, err := exec.LookPath(name); err == nil {
				whisperPath = p
				return
			}
		}
		for _, p := range []string{"/opt/homebrew/bin/whisper-cli", "/usr/local/bin/whisper-cli", "/opt/homebrew/bin/whisper-cpp"} {
			if _, err := os.Stat(p); err == nil {
				whisperPath = p
				return
			}
		}
	})
	return whisperPath
}

// ModelPath is where a model file lives in the cache.
func ModelPath(cacheDir, model string) string {
	return filepath.Join(cacheDir, "models", "ggml-"+model+".bin")
}

// EnsureModel downloads a model if it is missing. progress receives 0..1.
func EnsureModel(cacheDir, model string, progress func(float64)) (string, error) {
	p := ModelPath(cacheDir, model)
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 {
		return p, nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	u := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-" + model + ".bin"
	if strings.Contains(model, "tdrz") {
		u = "https://huggingface.co/akashmjn/tinydiarize-whisper.cpp/resolve/main/ggml-" + model + ".bin"
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", art.UserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("model download: HTTP %d", resp.StatusCode)
	}
	tmp := p + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	var done int64
	buf := make([]byte, 512<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return "", werr
			}
			done += int64(n)
			if progress != nil && resp.ContentLength > 0 {
				progress(float64(done) / float64(resp.ContentLength))
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return "", rerr
		}
	}
	f.Close()
	return p, os.Rename(tmp, p)
}

var pctRe = regexp.MustCompile(`progress\s*=\s*(\d+)%`)

// Options control a transcription run.
type Options struct {
	Bin      string // whisper binary (auto when empty)
	Model    string // model name
	CacheDir string
	Threads  int
	// Progress receives a phase name and 0..1.
	Progress func(phase string, frac float64)
}

// Transcribe runs whisper.cpp on an audio file and returns timed words
// grouped into captions. It converts the audio to 16 kHz mono WAV first.
func Transcribe(audio string, o Options) (*Transcript, error) {
	bin := FindWhisper(o.Bin)
	if bin == "" {
		return nil, errors.New("whisper.cpp not found: brew install whisper-cpp")
	}
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New("ffmpeg is needed to prepare audio for whisper")
	}
	report := func(ph string, f float64) {
		if o.Progress != nil {
			o.Progress(ph, f)
		}
	}
	model := o.Model
	if model == "" {
		model = "base.en"
	}
	report("model", 0)
	mp, err := EnsureModel(o.CacheDir, model, func(f float64) { report("model", f) })
	if err != nil {
		return nil, err
	}
	tmpDir, err := os.MkdirTemp("", "sopebox-whisper-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	wav := filepath.Join(tmpDir, "audio.wav")
	report("convert", 0)
	conv := exec.Command(ff, "-v", "quiet", "-nostdin", "-y", "-i", audio, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", wav)
	if out, err := conv.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %v %s", err, strings.TrimSpace(string(out)))
	}
	report("transcribe", 0)
	base := filepath.Join(tmpDir, "out")
	threads := o.Threads
	if threads <= 0 {
		threads = 4
	}
	args := []string{"-m", mp, "-f", wav, "-oj", "-of", base, "-ml", "1", "-sow", "-pp", "-t", strconv.Itoa(threads), "-np"}
	if strings.Contains(model, "tdrz") {
		args = append(args, "-tdrz")
	}
	cmd := exec.Command(bin, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go io.Copy(io.Discard, stdout)
	sc := bufio.NewScanner(stderr)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if m := pctRe.FindStringSubmatch(sc.Text()); m != nil {
			p, _ := strconv.Atoi(m[1])
			report("transcribe", float64(p)/100)
		}
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("whisper: %v", err)
	}
	raw, err := os.ReadFile(base + ".json")
	if err != nil {
		return nil, fmt.Errorf("whisper produced no output: %v", err)
	}
	t, err := ParseJSON(raw)
	if err != nil {
		return nil, err
	}
	t.Source = "whisper:" + model
	report("done", 1)
	return t, nil
}
