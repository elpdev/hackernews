package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/pkg/hn"
)

func (c Comments) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	headerText := c.headerText(width)
	headerLines := strings.Count(headerText, "\n")

	if len(c.order) == 0 {
		if c.loading == "" && c.err == "" {
			headerText += "No comments yet. Press esc to go back.\n"
		}
		return headerText
	}

	contentHeight := maxScreen(1, height-headerLines)
	lines, starts := c.renderComments(width)
	if len(lines) == 0 {
		return headerText
	}
	sel := clampIndex(c.selected, len(starts))
	cursor := starts[sel]
	maxTop := maxScreen(0, len(lines)-contentHeight)
	top := centeredTop(cursor, contentHeight, maxTop)
	end := minScreen(len(lines), top+contentHeight)

	var b strings.Builder
	b.WriteString(headerText)
	for i := top; i < end; i++ {
		line := lines[i]
		if i == cursor {
			line = articleLineHighlight(width).Render(padLine(ansi.Strip(line), width))
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (c Comments) headerText(width int) string {
	var head strings.Builder
	for _, line := range commentsHeaderBlock(c.story, width) {
		head.WriteString(line + "\n")
	}
	if c.loading != "" {
		head.WriteString(truncateScreen(c.loading, width) + "\n")
	}
	if c.err != "" {
		head.WriteString(truncateScreen(c.err, width) + "\n")
	}
	if c.status != "" {
		head.WriteString(truncateScreen(c.status, width) + "\n")
	}
	if c.searching || c.searchQuery != "" {
		label := "Filter"
		if c.searching {
			label = "Search"
		}
		query := c.searchQuery
		if query == "" {
			query = lipgloss.NewStyle().Faint(true).Render("type to search comments...")
		}
		head.WriteString(truncateScreen(label+": "+query, width) + "\n")
	}
	head.WriteString("\n")
	return head.String()
}

func (c Comments) renderComments(width int) ([]string, []int) {
	lines := make([]string, 0, len(c.order)*4)
	starts := make([]int, 0, len(c.order))
	muted := lipgloss.NewStyle().Faint(true)
	title := lipgloss.NewStyle().Bold(true)
	for _, line := range c.order {
		item := c.tree[line.id]
		indent := strings.Repeat("│ ", line.depth)
		starts = append(starts, len(lines))
		lines = append(lines, c.renderCommentHeader(item, line, indent, muted, title, width))

		if !c.collapsed[line.id] {
			body := commentBodyMarkdown(item)
			bodyWidth := maxScreen(20, width-lipgloss.Width(indent))
			rendered := renderMarkdown(body, bodyWidth)
			for _, bline := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				lines = append(lines, truncateScreen(muted.Render(indent)+bline, width))
			}
		}
		lines = append(lines, "")
	}
	return lines, starts
}

func (c Comments) renderCommentHeader(item hn.Item, line commentLine, indent string, muted, title lipgloss.Style, width int) string {
	var headerParts []string
	if item.Deleted || item.Dead {
		headerParts = append(headerParts, title.Render("[deleted]"))
	} else {
		author := item.By
		if author == "" {
			author = "anonymous"
		}
		headerParts = append(headerParts, title.Render("@"+author))
		if ts := relativeAge(item.Time); ts != "" {
			headerParts = append(headerParts, muted.Render(ts))
		}
	}
	replyCount := countDescendants(c.tree, line.id)
	if c.collapsed[line.id] && replyCount > 0 {
		headerParts = append(headerParts, muted.Render(fmt.Sprintf("[+%d hidden]", replyCount)))
	}
	return truncateScreen(muted.Render(indent)+strings.Join(headerParts, " · "), width)
}

func commentsHeaderBlock(story hn.Item, width int) []string {
	var out []string
	titleStyle := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Faint(true)
	if strings.TrimSpace(story.Title) != "" {
		out = append(out, truncateScreen(titleStyle.Render(story.Title), width))
	}
	if meta := commentsStoryMeta(story); meta != "" {
		out = append(out, truncateScreen(muted.Render(meta), width))
	}
	if strings.TrimSpace(story.Text) != "" {
		rendered := renderMarkdown(commentHTMLToMarkdown(story.Text), width)
		for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
			out = append(out, truncateScreen(line, width))
		}
	}
	out = append(out, truncateScreen(muted.Render("esc back · j/k move · left/right or p/n prev/next · / search · P parent · space collapse · o open"), width))
	return out
}

func commentsStoryMeta(story hn.Item) string {
	var metaParts []string
	if story.By != "" {
		metaParts = append(metaParts, "by "+story.By)
	}
	if story.Score > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d points", story.Score))
	}
	if story.Descendants > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d comments", story.Descendants))
	}
	if ts := relativeAge(story.Time); ts != "" {
		metaParts = append(metaParts, ts)
	}
	if story.URL != "" {
		if domain := storyDomain(story.URL); domain != "" {
			metaParts = append(metaParts, domain)
		}
	}
	return strings.Join(metaParts, " · ")
}
