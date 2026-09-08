package ui

import "github.com/Cid-Emmerich/SopeBox/internal/store"

// playRequest asks the UI loop to start an episode (used by scripted runs).
type playRequest struct{ item store.Item }
