package tool

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	uia "github.com/auuunya/go-element"
)

type IconCoord struct {
	Name   string
	Path   string
	X      int
	Y      int
	Width  int
	Height int
}

// desktopDirs returns the user and public desktop directories.
func desktopDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Desktop"))
		// Also check OneDrive desktop if it exists
		entries, _ := filepath.Glob(filepath.Join(home, "OneDrive*", "Desktop"))
		dirs = append(dirs, entries...)
	}
	dirs = append(dirs, `C:\Users\Public\Desktop`)
	return dirs
}

// resolveIconPath finds the actual file/folder path on disk for a desktop icon name.
func resolveIconPath(name string) string {
	for _, dir := range desktopDirs() {
		// Exact match (file or folder without extension)
		exact := filepath.Join(dir, name)
		if _, err := os.Stat(exact); err == nil {
			return exact
		}
		// Try common extensions: .lnk, .url, .exe, .pdf, .txt, .docx, .xlsx, etc.
		for _, ext := range []string{".lnk", ".url", ".exe", ".pdf", ".txt", ".docx", ".xlsx", ".pptx", ".zip", ".rar"} {
			p := filepath.Join(dir, name+ext)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		// Glob for any extension matching the name
		matches, _ := filepath.Glob(filepath.Join(dir, name+".*"))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func GetCoord() []IconCoord {
	var icons []IconCoord
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err := uia.CoInitialize()
	if err != nil {
		log.Fatalf("Error initializing COM: %v\n", err)
	}
	defer uia.CoUninitialize()

	instance, err := uia.CreateInstance(
		uia.CLSID_CUIAutomation,
		uia.IID_IUIAutomation,
		uia.CLSCTX_INPROC_SERVER,
	)
	if err != nil {
		log.Fatalf("Error creating UI Automation instance: %v\n", err)
	}
	ppv := uia.NewIUIAutomation(uia.NewIUnKnown(instance))

	root, err := ppv.GetRootElement()
	if err != nil {
		log.Fatalf("Error getting root element: %v\n", err)
	}

	progmanHwnd := uia.FindWindowA("Progman", "Program Manager")
	defViewHwnd := uintptr(0)

	if progmanHwnd != 0 {
		defViewHwnd = uia.FindWindowExA(progmanHwnd, 0, "SHELLDLL_DefView", "")
	}

	if defViewHwnd == 0 {
		var hwnd uintptr
		for {
			hwnd = uia.FindWindowExA(0, hwnd, "WorkerW", "")
			if hwnd == 0 {
				break
			}
			dv := uia.FindWindowExA(hwnd, 0, "SHELLDLL_DefView", "")
			if dv != 0 {
				defViewHwnd = dv
				break
			}
		}
	}

	if defViewHwnd == 0 {
		log.Fatal("Could not find SHELLDLL_DefView (desktop icon container)")
	}

	listViewHwnd := uia.FindWindowExA(defViewHwnd, 0, "SysListView32", "FolderView")
	if listViewHwnd == 0 {
		log.Fatal("Could not find SysListView32 (FolderView)")
	}

	listElem, err := uia.ElementFromHandle(ppv, listViewHwnd)
	if err != nil || listElem == nil {
		return getViaTraversal(ppv, root)
	}

	trueCondition := uia.CreateTrueCondition(ppv)
	childrenArray, err := listElem.FindAll(trueCondition)
	if err != nil {
		log.Fatalf("Error finding desktop items: %v\n", err)
	}
	if childrenArray == nil {
		log.Fatal("No desktop items found")
	}

	length := childrenArray.Get_Length()

	for i := int32(0); i < length; i++ {
		child, err := childrenArray.GetElement(i)
		if err != nil || child == nil {
			continue
		}

		elem := uia.NewElement(child)
		elem.Name()
		elem.BoundingRectangle()

		if elem.CurrentName == "" {
			continue
		}

		var x, y, w, h int32
		if elem.CurrentBoundingRectangle != nil {
			x = elem.CurrentBoundingRectangle.Left
			y = elem.CurrentBoundingRectangle.Top
			w = elem.CurrentBoundingRectangle.Right - elem.CurrentBoundingRectangle.Left
			h = elem.CurrentBoundingRectangle.Bottom - elem.CurrentBoundingRectangle.Top
		}

		icons = append(icons, IconCoord{
			Name:   elem.CurrentName,
			Path:   resolveIconPath(elem.CurrentName),
			X:      int(x),
			Y:      int(y),
			Width:  int(w),
			Height: int(h),
		})
	}
	return icons
}

func GetNameBasedSearch(name string) []IconCoord {
	coord := GetCoord()
	var result []IconCoord
	for _, c := range coord {
		if c.Name == name {
			result = append(result, c)
		}
	}
	return result
}

// getViaTraversal is a fallback that walks the full UI tree to find desktop icons.
func getViaTraversal(ppv *uia.IUIAutomation, root *uia.IUIAutomationElement) []IconCoord {
	tree := uia.TraverseUIElementTree(ppv, root)

	listViewElem := uia.SearchElem(tree, func(e *uia.Element) bool {
		return e.CurrentClassName == "SysListView32" && e.CurrentName == "FolderView"
	})
	if listViewElem == nil {
		listViewElem = uia.SearchElem(tree, func(e *uia.Element) bool {
			return e.CurrentClassName == "SysListView32"
		})
	}
	if listViewElem == nil {
		log.Fatal("Could not find desktop icon list via tree traversal")
	}

	var icons []IconCoord
	for _, child := range listViewElem.Child {
		if child.CurrentName == "" {
			continue
		}

		rect := child.CurrentBoundingRectangle
		var x, y, w, h int32
		if rect != nil {
			x = rect.Left
			y = rect.Top
			w = rect.Right - rect.Left
			h = rect.Bottom - rect.Top
		}

		icons = append(icons, IconCoord{
			Name:   child.CurrentName,
			Path:   resolveIconPath(child.CurrentName),
			X:      int(x),
			Y:      int(y),
			Width:  int(w),
			Height: int(h),
		})
	}
	return icons
}
