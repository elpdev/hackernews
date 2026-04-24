package screens

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/media"
)

func (s Saved) articleView(width, height int) string {
	item, ok := s.itemByID(s.readID)
	if !ok {
		return "Saved article not found. Press esc to go back."
	}
	header := []string{"esc back | s/d delete | o open | y copy | j/k line | left/right or p/n paragraph"}
	if s.status != "" {
		header = append(header, s.status)
	}
	contentHeight := maxScreen(1, height-len(header)-1)
	contentWidth := articleContentWidth(width)
	lines := renderedArticleLines(item.ID, contentWidth, item.Article, articleImage{}, nil)
	maxTop := maxScreen(0, len(lines)-contentHeight)
	cursor := clampIndex(s.readLine, len(lines))
	top := centeredTop(cursor, contentHeight, maxTop)
	end := minScreen(len(lines), top+contentHeight)

	var b strings.Builder
	for _, line := range header {
		b.WriteString(truncateScreen(line, width) + "\n")
	}
	for i := top; i < end; i++ {
		line := lines[i]
		if i == cursor && !containsInlineImage(line) {
			line = articleLineHighlight(contentWidth).Render(padLine(ansi.Strip(line), contentWidth))
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return media.ViewportPrefix() + b.String()
}

func centeredTop(cursor, contentHeight, maxTop int) int {
	top := cursor - contentHeight/2
	if top < 0 {
		return 0
	}
	if top > maxTop {
		return maxTop
	}
	return top
}
