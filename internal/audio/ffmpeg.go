package audio

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ffmpegOnce sync.Once
	ffmpegPath string
)

// HaveFFmpeg reports whether ffmpeg is on PATH.
func HaveFFmpeg() bool {
	ffmpegOnce.Do(func() {
		ffmpegPath, _ = exec.LookPath("ffmpeg")
	})
	return ffmpegPath != ""
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// streamer decodes a file or URL through ffmpeg into 32-bit float stereo
// PCM. Speed changes use ffmpeg's atempo filter so pitch is preserved.
// A background goroutine reads the pipe into a ring so network hiccups
// do not block the audio thread.
type streamer struct {
	source string
	length float64 // seconds, 0 if unknown
	speed  float64

	mu     sync.Mutex
	cmd    *exec.Cmd
	base   float64 // position the decoder was started at
	pos    int     // output samples consumed since base (at output rate)
	buf    [][2]float64
	head   int // read index
	count  int // samples in ring
	eof    bool
	err    error
	closed bool
	gen    int
}

const ringSeconds = 8

func open(source string, start, speed, duration float64) (*streamer, error) {
	if !HaveFFmpeg() {
		return nil, errors.New("ffmpeg is required to play podcasts (brew install ffmpeg)")
	}
	s := &streamer{source: source, length: duration, speed: speed}
	s.buf = make([][2]float64, int(SampleRate)*ringSeconds)
	if s.length <= 0 && !isURL(source) {
		s.length = ProbeDuration(source)
	}
	if err := s.start(start, speed); err != nil {
		return nil, err
	}
	if isURL(source) && s.length <= 0 {
		go func() {
			if d := ProbeDuration(source); d > 0 {
				s.mu.Lock()
				s.length = d
				s.mu.Unlock()
			}
		}()
	}
	return s, nil
}

func tempoFilter(speed float64) string {
	// atempo accepts 0.5..100 per stage in modern ffmpeg but chain for safety
	var parts []string
	for speed > 2 {
		parts = append(parts, "atempo=2")
		speed /= 2
	}
	for speed < 0.5 {
		parts = append(parts, "atempo=0.5")
		speed *= 2
	}
	parts = append(parts, "atempo="+strconv.FormatFloat(speed, 'f', 3, 64))
	return strings.Join(parts, ",")
}

func (s *streamer) start(at, speed float64) error {
	s.stopLocked()
	s.gen++
	gen := s.gen
	args := []string{"-v", "quiet", "-nostdin"}
	if isURL(s.source) {
		args = append(args, "-reconnect", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "5",
			"-user_agent", "SopeBox/1.0 (terminal podcast player)")
	}
	if at > 0 {
		args = append(args, "-ss", strconv.FormatFloat(at, 'f', 3, 64))
	}
	args = append(args, "-i", s.source, "-vn")
	if math.Abs(speed-1) > 0.001 {
		args = append(args, "-filter:a", tempoFilter(speed))
	}
	args = append(args, "-f", "f32le", "-acodec", "pcm_f32le", "-ac", "2", "-ar", strconv.Itoa(int(SampleRate)), "-")
	cmd := exec.Command(ffmpegPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}
	s.cmd = cmd
	s.base = at
	s.pos = 0
	s.head, s.count = 0, 0
	s.eof = false
	s.speed = speed
	go s.pump(bufio.NewReaderSize(stdout, 1<<16), gen)
	return nil
}

// pump reads PCM from ffmpeg into the ring buffer.
func (s *streamer) pump(r *bufio.Reader, gen int) {
	chunk := make([]byte, 8*1024)
	for {
		n, err := io.ReadFull(r, chunk)
		if n > 0 {
			frames := n / 8
			s.mu.Lock()
			if s.gen != gen || s.closed {
				s.mu.Unlock()
				return
			}
			for len(s.buf)-s.count < frames {
				// ring is full: wait for the consumer
				s.mu.Unlock()
				time.Sleep(20 * time.Millisecond)
				s.mu.Lock()
				if s.gen != gen || s.closed {
					s.mu.Unlock()
					return
				}
			}
			for i := 0; i < frames; i++ {
				l := math.Float32frombits(binary.LittleEndian.Uint32(chunk[i*8 : i*8+4]))
				rr := math.Float32frombits(binary.LittleEndian.Uint32(chunk[i*8+4 : i*8+8]))
				idx := (s.head + s.count) % len(s.buf)
				s.buf[idx] = [2]float64{float64(l), float64(rr)}
				s.count++
			}
			s.mu.Unlock()
		}
		if err != nil {
			s.mu.Lock()
			if s.gen == gen {
				s.eof = true
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					s.err = err
				}
			}
			s.mu.Unlock()
			return
		}
	}
}

func (s *streamer) stopLocked() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		go s.cmd.Wait()
	}
	s.cmd = nil
}

// Stream pulls samples from the ring for the audio thread.
func (s *streamer) Stream(samples [][2]float64) (n int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, false
	}
	for n < len(samples) && s.count > 0 {
		samples[n] = s.buf[s.head]
		s.head = (s.head + 1) % len(s.buf)
		s.count--
		n++
		s.pos++
	}
	if n < len(samples) && s.eof && s.count == 0 {
		return n, n > 0
	}
	return n, true
}

// starved reports whether the ring ran dry before EOF (network stall).
func (s *streamer) starved() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.eof && s.count == 0
}

func (s *streamer) buffered() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return float64(s.count) / float64(SampleRate)
}

func (s *streamer) Err() error { s.mu.Lock(); defer s.mu.Unlock(); return s.err }

// Position in seconds of source time.
func (s *streamer) Position() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.base + float64(s.pos)/float64(SampleRate)*s.speed
}

// Seek restarts decoding at pos seconds with the given speed.
func (s *streamer) Seek(pos, speed float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pos < 0 {
		pos = 0
	}
	_ = s.start(pos, speed)
}

// Close kills ffmpeg.
func (s *streamer) Close() {
	s.mu.Lock()
	s.closed = true
	s.stopLocked()
	s.mu.Unlock()
}

// ProbeDuration asks ffprobe for the length of a file or URL in seconds.
func ProbeDuration(path string) float64 {
	probe, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0
	}
	args := []string{"-v", "quiet"}
	if isURL(path) {
		args = append(args, "-user_agent", "SopeBox/1.0")
	}
	args = append(args, "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	cmd := exec.Command(probe, args...)
	done := make(chan struct{})
	var out []byte
	go func() {
		out, _ = cmd.Output()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return 0
	}
	d, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return d
}
