package vis

import (
	"strings"
	"testing"
)

func TestRegistry(t *testing.T) {
	got := strings.Join(Names(), ",")
	if got != "constellation,wordflow,talktime,bars" {
		t.Fatalf("styles = %s", got)
	}
	// a config saved with a style that no longer exists falls back to the first
	for _, old := range []string{"orbs", "halo", "ribbon", "pulse", "collage"} {
		if Index(old) != 0 {
			t.Errorf("%s: index %d", old, Index(old))
		}
	}
}
