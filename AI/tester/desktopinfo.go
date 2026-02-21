package main

import tool "desktop_agent/tools/desktopinfo"

func main() {
	icons := tool.GetNameBasedSearch("HW1")
	for _, icon := range icons {
		println("Icon:", icon.Name, "\nPath:", icon.Path, "\nX:", icon.X, "\nY:", icon.Y, "\nWidth:", icon.Width, "\nHeight:", icon.Height)
	}
}
