package operations

import (
	"encoding/json"
	"strings"
	"testing"
)

func navigationFixture() Navigation {
	return Navigation{Version: 1,
		Header: []NavigationItem{
			{Section: "features", Menu: "auto"},
			{Section: "use-cases", Label: NavigationCopy{EN: " Solutions ", ZH: "方案"}, Menu: "manual", Items: []string{"/use-cases/subscription-saas", "/compare/starter-vs-from-scratch"}},
			{Section: "pricing", Menu: "none"},
		},
		Footer: []NavigationGroup{{Title: NavigationCopy{EN: "Product", ZH: "产品"}, Links: []string{"features", "pricing"}}},
	}
}

func TestNavigationAcceptsKnownSectionsAndArticlePathsOnly(t *testing.T) {
	valid := navigationFixture()
	if !validNavigation(&valid) || valid.Header[1].Label.EN != "Solutions" || valid.Header[0].Items == nil {
		t.Fatalf("valid navigation rejected or not normalized: %+v", valid)
	}
	for name, change := range map[string]func(*Navigation){
		"version":               func(n *Navigation) { n.Version = 2 },
		"unknown section":       func(n *Navigation) { n.Header[0].Section = "admin" },
		"duplicate section":     func(n *Navigation) { n.Header[2].Section = "features" },
		"unknown menu":          func(n *Navigation) { n.Header[0].Menu = "all" },
		"menu on a plain page":  func(n *Navigation) { n.Header[2].Menu = "auto" },
		"external item":         func(n *Navigation) { n.Header[1].Items = []string{"https://example.com/docs/x"} },
		"private item":          func(n *Navigation) { n.Header[1].Items = []string{"/admin/users"} },
		"traversal item":        func(n *Navigation) { n.Header[1].Items = []string{"/docs/../admin"} },
		"duplicate item":        func(n *Navigation) { n.Header[1].Items = []string{"/docs/a", "/docs/a"} },
		"empty manual menu":     func(n *Navigation) { n.Header[1].Items = nil },
		"items without manual":  func(n *Navigation) { n.Header[0].Items = []string{"/features/x"} },
		"long label":            func(n *Navigation) { n.Header[0].Label.ZH = strings.Repeat("长", 41) },
		"multiline label":       func(n *Navigation) { n.Header[0].Label.EN = "a\nb" },
		"untitled footer group": func(n *Navigation) { n.Footer[0].Title.ZH = " " },
		"unknown footer link":   func(n *Navigation) { n.Footer[0].Links = []string{"/account"} },
		"repeated footer link": func(n *Navigation) {
			n.Footer = append(n.Footer, NavigationGroup{Title: NavigationCopy{EN: "More", ZH: "更多"}, Links: []string{"pricing"}})
		},
		"too many header items": func(n *Navigation) {
			for _, section := range []string{"docs", "blog", "about", "contact"} {
				n.Header = append(n.Header, NavigationItem{Section: section, Menu: "none"})
			}
		},
	} {
		n := navigationFixture()
		change(&n)
		if validNavigation(&n) {
			t.Errorf("%s: accepted %+v", name, n)
		}
	}
	empty := Navigation{Version: 1}
	if !validNavigation(&empty) {
		t.Fatal("an empty header and footer is a valid choice")
	}
}

func TestNavigationIsSavedAuditedAndPublic(t *testing.T) {
	f := newFixture(t)
	if w := request(f, "GET", "/site/settings", "", "", ""); w.Code != 200 || data(t, w)["navigation"] != nil {
		t.Fatalf("unsaved navigation must be null so the site uses its defaults: %s", w.Body)
	}
	raw, _ := json.Marshal(map[string]any{"navigation": navigationFixture(), "reason": "Simplify the header"})
	if w := request(f, "PATCH", "/admin/settings", string(raw), "admin", "navigation-1"); w.Code != 200 {
		t.Fatalf("navigation update: %s", w.Body)
	}
	w := request(f, "GET", "/site/settings", "", "", "")
	var public struct {
		Data struct {
			BrandName  string      `json:"brand_name"`
			Navigation *Navigation `json:"navigation"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &public); err != nil || public.Data.Navigation == nil {
		t.Fatalf("public navigation: %s", w.Body)
	}
	if got := public.Data.Navigation.Header[1]; got.Label.EN != "Solutions" || len(got.Items) != 2 || got.Items[1] != "/compare/starter-vs-from-scratch" {
		t.Fatalf("saved header item: %+v", got)
	}
	if public.Data.BrandName != DefaultSettings().BrandName {
		t.Fatalf("a navigation-only patch must not change other settings: %s", w.Body)
	}
	var audit AuditEvent
	if err := f.db.Where("action = ? AND status = ?", "settings.update", "succeeded").Take(&audit).Error; err != nil || !strings.Contains(audit.Details, "/compare/starter-vs-from-scratch") {
		t.Fatalf("navigation audit: %+v %v", audit, err)
	}
	bad, _ := json.Marshal(map[string]any{"navigation": map[string]any{"version": 1, "header": []any{map[string]any{"section": "features", "menu": "manual", "items": []string{"https://example.com"}}}, "footer": []any{}}, "reason": "Link out"})
	if w := request(f, "PATCH", "/admin/settings", string(bad), "admin", "navigation-2"); w.Code != 400 {
		t.Fatalf("external menu link accepted: %s", w.Body)
	}
}
