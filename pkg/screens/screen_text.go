package screens

func clampIndex(idx, length int) int {
	if length == 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}

func truncateScreen(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width == 1 {
		return "."
	}
	return s[:width-1] + "."
}

func minScreen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxScreen(a, b int) int {
	if a > b {
		return a
	}
	return b
}
