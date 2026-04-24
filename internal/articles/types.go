package articles

type Article struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	URL      string `json:"url"`
	Image    string `json:"image"`
	Excerpt  string `json:"excerpt"`
	Markdown string `json:"markdown"`
}

type Extractor interface {
	Extract(url string) (Article, error)
}
