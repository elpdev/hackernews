package screens

import "github.com/elpdev/hackernews/internal/hn"

// OpenCommentsMsg is emitted by a stories screen to request drilling into a
// story's comment thread. The app dispatcher activates the comments screen and
// routes back to ReturnTo when the user leaves.
type OpenCommentsMsg struct {
	Story    hn.Item
	ReturnTo string
}

// NavigateMsg is emitted by a screen to request activation of a sibling screen
// by ID (e.g. Comments leaving back to the originating feed).
type NavigateMsg struct {
	ScreenID string
}
