package screens

import "charm.land/glamour/v2"

func renderMarkdown(markdown string, width int) string {
	markdown = normalizeMarkdownForRender(markdown)
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(maxScreen(20, width)),
	)
	if err != nil {
		return markdown
	}
	out, err := r.Render(markdown)
	if err != nil {
		return markdown
	}
	return out
}

func normalizeMarkdownForRender(markdown string) string {
	markdown = repairLooseListItems(markdown)
	markdown = fenceLooseArticleCode(markdown)
	return labelUnlabeledCodeFences(markdown)
}
