package workspace

import "time"

// nowUTC is a seam for tests; truncated to seconds because sub-second manifest
// timestamps are noise in diffs.
var nowUTC = func() time.Time { return time.Now().UTC().Truncate(time.Second) }
