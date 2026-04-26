package hn

import "time"

type Item struct {
	ID          int    `json:"id"`
	Deleted     bool   `json:"deleted"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Text        string `json:"text"`
	Dead        bool   `json:"dead"`
	Parent      int    `json:"parent"`
	Kids        []int  `json:"kids"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Title       string `json:"title"`
	Descendants int    `json:"descendants"`
}

func (i Item) CreatedAt() time.Time {
	if i.Time <= 0 {
		return time.Time{}
	}
	return time.Unix(i.Time, 0)
}

func (i Item) Readable() bool {
	if i.Deleted || i.Dead {
		return false
	}
	switch i.Type {
	case "story", "job", "poll":
		return true
	}
	return false
}
