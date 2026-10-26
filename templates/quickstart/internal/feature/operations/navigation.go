package operations

import (
	"strings"
	"unicode/utf8"

	"github.com/brizenchi/quickstart-template/internal/feature/content"
)

// Navigation is the public header and footer chosen in admin. A nil value keeps
// the frontend defaults. Destinations are known site sections and article paths
// only, so a saved menu cannot point outside the site or to an arbitrary URL.
type Navigation struct {
	Version int               `json:"version"`
	Header  []NavigationItem  `json:"header"`
	Footer  []NavigationGroup `json:"footer"`
}

type NavigationCopy struct {
	EN string `json:"en"`
	ZH string `json:"zh"`
}

// NavigationItem is one header destination. Menu "auto" lists the section's
// published pages, "manual" lists Items in order, and "none" shows no menu.
// An empty label uses the frontend's default name for the section.
type NavigationItem struct {
	Section string         `json:"section"`
	Label   NavigationCopy `json:"label"`
	Menu    string         `json:"menu"`
	Items   []string       `json:"items"`
}

// NavigationGroup is one footer column of section links.
type NavigationGroup struct {
	Title NavigationCopy `json:"title"`
	Links []string       `json:"links"`
}

const (
	maxHeaderItems    = 6
	maxMenuItems      = 10
	maxFooterGroups   = 4
	maxGroupLinks     = 8
	maxNavigationCopy = 40
)

// Sections whose directory lists articles and can therefore open a menu.
var menuSections = map[string]bool{"features": true, "use-cases": true, "compare": true, "docs": true, "blog": true, "resources": true}

// Every section a header item or footer link may name.
var navigationSections = map[string]bool{
	"features": true, "use-cases": true, "compare": true, "docs": true, "blog": true, "resources": true,
	"updates": true, "pricing": true, "about": true, "contact": true,
}

// validNavigation trims labels and normalizes Items, then checks every limit.
func validNavigation(n *Navigation) bool {
	if n.Version != 1 || len(n.Header) > maxHeaderItems || len(n.Footer) > maxFooterGroups {
		return false
	}
	header := map[string]bool{}
	for i := range n.Header {
		item := &n.Header[i]
		if !navigationSections[item.Section] || header[item.Section] || !validNavigationCopy(&item.Label, false) {
			return false
		}
		header[item.Section] = true
		switch item.Menu {
		case "auto":
			if !menuSections[item.Section] || len(item.Items) != 0 {
				return false
			}
			item.Items = []string{}
		case "none":
			if len(item.Items) != 0 {
				return false
			}
			item.Items = []string{}
		case "manual":
			if !menuSections[item.Section] || len(item.Items) == 0 || len(item.Items) > maxMenuItems {
				return false
			}
			paths := map[string]bool{}
			for _, path := range item.Items {
				if !content.ValidArticlePath(path) || paths[path] {
					return false
				}
				paths[path] = true
			}
		default:
			return false
		}
	}
	footer := map[string]bool{}
	for i := range n.Footer {
		group := &n.Footer[i]
		if !validNavigationCopy(&group.Title, true) || len(group.Links) == 0 || len(group.Links) > maxGroupLinks {
			return false
		}
		for _, link := range group.Links {
			if !navigationSections[link] || footer[link] {
				return false
			}
			footer[link] = true
		}
	}
	return true
}

func validNavigationCopy(copy *NavigationCopy, required bool) bool {
	for _, value := range []*string{&copy.EN, &copy.ZH} {
		*value = strings.TrimSpace(*value)
		if !utf8.ValidString(*value) || utf8.RuneCountInString(*value) > maxNavigationCopy || strings.ContainsAny(*value, "\x00\r\n") || (required && *value == "") {
			return false
		}
	}
	return true
}
