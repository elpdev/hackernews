package screens

import (
	"fmt"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/media"
)

var articleRenderCache = struct {
	sync.Mutex
	lines map[string][]string
}{lines: make(map[string][]string)}

type articleImage struct {
	url   string
	bytes []byte
	err   string
}

func (t Top) articleView(width, height int) string {
	article := t.articles[t.readID]
	if t.imageURL != "" {
		return t.articleImageView(width, height, article)
	}
	saveHelp := "s save"
	if t.savedIDs[t.readID] {
		saveHelp = "s unsave"
	}
	header := []string{"esc back | " + saveHelp + " | o open in browser | y copy url | j/k move | [/ ] paragraph"}
	header[0] = "esc back | " + saveHelp + " | o open | y copy | i image | j/k line | left/right or p/n paragraph"
	if t.err != "" {
		header = append(header, t.err)
	}
	if t.status != "" {
		header = append(header, t.status)
	}
	contentHeight := maxScreen(1, height-len(header)-1)
	contentWidth := articleContentWidth(width)
	lines := renderedArticleLines(t.readID, contentWidth, article, t.images[t.readID], t.bodyImages[t.readID])
	maxTop := maxScreen(0, len(lines)-contentHeight)
	cursor := clampIndex(t.readLine, len(lines))
	top := articleViewportTop(cursor, contentHeight, maxTop)
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

func articleViewportTop(cursor, contentHeight, maxTop int) int {
	if contentHeight <= 0 || maxTop <= 0 {
		return 0
	}
	step := maxScreen(1, contentHeight-4)
	top := (cursor / step) * step
	if top > maxTop {
		return maxTop
	}
	return top
}

func (t Top) articleImageView(width, height int, _ articles.Article) string {
	header := []string{"esc close image | o open image | y copy image url"}
	contentHeight := maxScreen(1, height-len(header)-1)
	image := articleImage{}
	if images := t.bodyImages[t.readID]; images != nil {
		image = images[t.imageURL]
	}
	if image.url == "" {
		image = articleImage{url: t.imageURL}
	}
	imageWidth := minScreen(maxScreen(12, articleContentWidth(width)-6), 72)
	block := articleImageBlockForURL(t.imageURL, image, imageWidth)
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}
	var b strings.Builder
	for _, line := range header {
		b.WriteString(truncateScreen(line, width) + "\n")
	}
	b.WriteString("Image: " + t.imageURL + "\n\n")
	b.WriteString(strings.Join(lines, "\n"))
	return media.ViewportPrefix() + b.String()
}

func articleContentWidth(width int) int {
	return minScreen(width, articleMaxWidth)
}

func containsInlineImage(line string) bool {
	return strings.Contains(line, "\x1b_G") || strings.Contains(line, "]1337;") || strings.Contains(line, "\x1bP1;1q")
}

func articleLineHighlight(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FDE68A")).
		Background(lipgloss.Color("#334155")).
		MaxWidth(maxScreen(0, width))
}

func padLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	return line + strings.Repeat(" ", width-lineWidth)
}

func renderedArticleLines(id, width int, article articles.Article, image articleImage, _ map[string]articleImage) []string {
	key := fmt.Sprintf("%d:%d:%s", id, width, image.cacheKey())
	articleRenderCache.Lock()
	if lines, ok := articleRenderCache.lines[key]; ok {
		articleRenderCache.Unlock()
		return lines
	}
	articleRenderCache.Unlock()

	rendered := renderArticle(article, image, width)
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	articleRenderCache.Lock()
	articleRenderCache.lines[key] = lines
	articleRenderCache.Unlock()
	return lines
}

func (i articleImage) cacheKey() string {
	if i.url == "" {
		return "none"
	}
	if len(i.bytes) > 0 {
		return fmt.Sprintf("loaded:%s:%d", i.url, len(i.bytes))
	}
	if i.err != "" {
		return "err:" + i.url
	}
	return "loading:" + i.url
}

func clearArticleRenderCache(id int) {
	prefix := fmt.Sprintf("%d:", id)
	articleRenderCache.Lock()
	defer articleRenderCache.Unlock()
	for key := range articleRenderCache.lines {
		if strings.HasPrefix(key, prefix) {
			delete(articleRenderCache.lines, key)
		}
	}
}

func cachedArticleLineCount(id int) int {
	prefix := fmt.Sprintf("%d:", id)
	articleRenderCache.Lock()
	defer articleRenderCache.Unlock()
	for key, lines := range articleRenderCache.lines {
		if strings.HasPrefix(key, prefix) {
			return len(lines)
		}
	}
	return 0
}

func cachedArticleLines(id int) []string {
	prefix := fmt.Sprintf("%d:", id)
	articleRenderCache.Lock()
	defer articleRenderCache.Unlock()
	for key, lines := range articleRenderCache.lines {
		if strings.HasPrefix(key, prefix) {
			return append([]string(nil), lines...)
		}
	}
	return nil
}

func nextParagraphLine(lines []string, cursor int) int {
	if len(lines) == 0 {
		return cursor
	}
	cursor = clampIndex(cursor, len(lines))
	for i := cursor + 1; i < len(lines); i++ {
		if isBlankRenderedLine(lines[i-1]) && !isBlankRenderedLine(lines[i]) {
			return i
		}
	}
	return len(lines) - 1
}

func previousParagraphLine(lines []string, cursor int) int {
	if len(lines) == 0 {
		return cursor
	}
	cursor = clampIndex(cursor, len(lines))
	for i := cursor - 1; i > 0; i-- {
		if isBlankRenderedLine(lines[i-1]) && !isBlankRenderedLine(lines[i]) {
			return i
		}
	}
	return 0
}

func isBlankRenderedLine(line string) bool {
	return strings.TrimSpace(ansi.Strip(line)) == ""
}

func renderArticle(article articles.Article, image articleImage, width int) string {
	sections := make([]string, 0, 4)
	if article.Title != "" {
		sections = append(sections, renderMarkdown("# "+article.Title+"\n", width))
	}
	if block := articleImageReferenceBlock(article, width); block != "" {
		sections = append(sections, block)
	}
	hasMeta := false
	if meta := articleMeta(article); meta != "" {
		sections = append(sections, renderMarkdown("> "+meta+"\n", width))
		hasMeta = true
	}
	hasBody := strings.TrimSpace(article.Markdown) != ""
	if hasBody {
		sections = append(sections, renderArticleBody(article, width))
	} else if url := strings.TrimSpace(article.URL); url != "" {
		sections = append(sections, renderMarkdown(articleFallbackBody(url), width))
	}
	trimmed := trimRenderedSections(sections)
	if hasMeta && hasBody && len(trimmed) >= 2 {
		return strings.Join(trimmed[:len(trimmed)-1], "\n") + "\n\n" + trimmed[len(trimmed)-1]
	}
	return strings.Join(trimmed, "\n")
}

func articleFallbackBody(url string) string {
	return "---\n\n" +
		"Couldn't extract readable content from this page.\n\n" +
		"Press `o` to open in your browser:\n\n" +
		url + "\n"
}

func articleImageBlock(article articles.Article, image articleImage, width int) string {
	imageURL := strings.TrimSpace(article.Image)
	if imageURL == "" {
		return ""
	}
	return articleImageBlockForURL(imageURL, image, width)
}

func articleImageReferenceBlock(article articles.Article, width int) string {
	imageURL := resolveArticleImageURL(article)
	if imageURL == "" {
		return ""
	}
	return renderMarkdown("**Image 1:** "+imageURL, width)
}

func articleImageBlockForURL(imageURL string, image articleImage, width int) string {
	if len(image.bytes) == 0 {
		if image.err == "" {
			return "Image: loading..."
		}
		if image.url != "" {
			return "Image: " + image.url
		}
		return "Image: " + imageURL
	}
	imageWidth := minScreen(maxScreen(12, width-6), 48)
	block, _, err := media.RenderBytes(image.bytes, imageWidth)
	if err != nil || block == "" {
		return "Image: " + imageURL
	}
	return block
}

func renderArticleBody(article articles.Article, width int) string {
	markdown := article.Markdown
	matches := markdownImagePattern.FindAllStringSubmatchIndex(markdown, -1)
	if len(matches) == 0 {
		return renderMarkdown(markdown, width)
	}

	sections := make([]string, 0, len(matches)*2+1)
	pos := 0
	imageNumber := 1
	if resolveArticleImageURL(article) != "" {
		imageNumber = 2
	}
	for _, match := range matches {
		if match[0] > pos {
			if text := strings.TrimSpace(markdown[pos:match[0]]); text != "" {
				sections = append(sections, renderMarkdown(text, width))
			}
		}
		rawURL := markdown[match[2]:match[3]]
		imageURL := resolveImageURL(rawURL, article.URL)
		if imageURL == "" {
			imageURL = rawURL
		}
		sections = append(sections, renderMarkdown(fmt.Sprintf("**Image %d:** %s", imageNumber, imageURL), width))
		imageNumber++
		pos = match[1]
	}
	if pos < len(markdown) {
		if text := strings.TrimSpace(markdown[pos:]); text != "" {
			sections = append(sections, renderMarkdown(text, width))
		}
	}
	return strings.Join(trimRenderedSections(sections), "\n\n")
}

func trimRenderedSections(sections []string) []string {
	trimmed := sections[:0]
	for _, section := range sections {
		section = strings.Trim(section, "\n")
		if section != "" {
			trimmed = append(trimmed, section)
		}
	}
	return trimmed
}

func articleMeta(article articles.Article) string {
	parts := make([]string, 0, 4)
	if article.Author != "" {
		parts = append(parts, "by "+article.Author)
	}
	if article.Date != "" {
		parts = append(parts, article.Date)
	}
	if article.URL != "" {
		parts = append(parts, article.URL)
	}
	return strings.Join(parts, " | ")
}
