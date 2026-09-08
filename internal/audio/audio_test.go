package audio

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// makeTone writes a 3 second 440 Hz mp3 with ffmpeg.
func makeTone(t *testing.T) string {
	t.Helper()
	if !HaveFFmpeg() {
		t.Skip("ffmpeg not installed")
	}
	p := filepath.Join(t.TempDir(), "tone.mp3")
	cmd := exec.Command("ffmpeg", "-v", "quiet", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=3", "-ac", "2", p)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg failed: %v %s", err, out)
	}
	return p
}

func drain(s *streamer, n int) (float64, int) {
	buf := make([][2]float64, 1024)
	got := 0
	peak := 0.0
	for got < n {
		k, ok := s.Stream(buf)
		for i := 0; i < k; i++ {
			peak = math.Max(peak, math.Abs(buf[i][0]))
		}
		got += k
		if !ok {
			break
		}
		if k == 0 {
			// wait for the pump
			for i := 0; i < 200 && s.buffered() == 0 && !s.eof; i++ {
				sleepMs(5)
			}
			if s.eof && s.buffered() == 0 {
				break
			}
		}
	}
	return peak, got
}

func TestStreamerDecodesSeeksAndSpeeds(t *testing.T) {
	p := makeTone(t)
	s, err := open(p, 0, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.length < 2.5 || s.length > 3.5 {
		t.Errorf("length = %v", s.length)
	}
	peak, n := drain(s, 44100)
	if n < 44100 || peak < 0.02 {
		t.Fatalf("decoded %d samples, peak %v", n, peak)
	}
	if pos := s.Position(); pos < 0.9 || pos > 1.2 {
		t.Errorf("position after 1s = %v", pos)
	}
	s.Seek(2, 1)
	drain(s, 22050)
	if pos := s.Position(); pos < 2.4 || pos > 2.7 {
		t.Errorf("position after seek+0.5s = %v", pos)
	}
	// double speed: half a second of output covers a second of source
	s.Seek(0, 2)
	drain(s, 22050)
	if pos := s.Position(); pos < 0.9 || pos > 1.2 {
		t.Errorf("position at 2x = %v", pos)
	}
	// run to the end
	_, rest := drain(s, 10*44100)
	if rest > 44100*2 {
		t.Errorf("too many samples after 2x: %d", rest)
	}
	if !s.eof {
		t.Error("no eof")
	}
}

func TestOpenMissing(t *testing.T) {
	if !HaveFFmpeg() {
		t.Skip("ffmpeg not installed")
	}
	s, err := open(filepath.Join(os.TempDir(), "definitely-missing.mp3"), 0, 1, 0)
	if err != nil {
		return
	}
	defer s.Close()
	buf := make([][2]float64, 256)
	for i := 0; i < 400; i++ {
		if _, ok := s.Stream(buf); !ok || (s.eof && s.buffered() == 0) {
			return
		}
		sleepMs(5)
	}
	t.Error("missing file never reported eof")
}
