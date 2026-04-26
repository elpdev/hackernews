package layout

import "github.com/elpdev/tuilayout"

func Calculate(width, height int, showSidebar bool) Dimensions {
	return tuilayout.Calculate(width, height, tuilayout.Options{ShowSidebar: showSidebar})
}
