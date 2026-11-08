package service

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// The album page is a self-contained HTML document the AI art director writes
// from scratch for one wedding. It is untrusted output: it is checked here
// before storage and only ever rendered inside a sandboxed, opaque-origin
// iframe whose security policy blocks the network (see the frontend's
// src/lib/album/signature/assemble.ts).
//
// Ported from VoyaLoom (internal/signature/validate.go), keeping only the
// album-runtime contract: every page is built on the runtime's markup.

const (
	// albumPhotoScheme is the placeholder the model uses for every photo. The
	// renderer swaps it for a fresh presigned URL, so stored pages never embed
	// expiring links and can only show this wedding's photos.
	albumPhotoScheme = "mv-photo://"

	// albumPhotosGlobal is injected by the renderer with every photo, which
	// the runtime's contact sheet shows.
	albumPhotosGlobal = "MOONVEILS_PHOTOS"

	maxAlbumPageBytes = 450_000
)

// albumPageRules describe what a generated page must reference.
type albumPageRules struct {
	// KnownPhotoIDs is every photo the page may reference.
	KnownPhotoIDs map[string]struct{}
	// RequiredPhotoIDs must appear (the emotional peak).
	RequiredPhotoIDs []string
	MinFeatured      int
	// MaxVisibleWords keeps the page an album rather than an article. Zero
	// disables the check.
	MaxVisibleWords int
	// Subjects holds, for photos with people in them, where the subject is
	// and where a title can go instead.
	Subjects map[string]photoSubject
}

// photoSubject is where a photo's people are, for keeping titles off them.
type photoSubject struct {
	FocalX, FocalY float64
	TextArea       string
}

// coversSubject reports whether a centred title would sit over the people:
// their focal point is near the middle of the frame.
func (p photoSubject) coversSubject() bool {
	dx, dy := p.FocalX-0.5, p.FocalY-0.5
	return dx > -0.2 && dx < 0.2 && dy > -0.25 && dy < 0.25
}

// allowedPageHosts are the only external origins a page may reference. The
// renderer's Content-Security-Policy enforces the same list in the browser;
// this check exists so the model can repair the page before it is stored.
var allowedPageHosts = map[string]struct{}{
	"fonts.googleapis.com": {},
	"fonts.gstatic.com":    {},
	"cdnjs.cloudflare.com": {},
	"cdn.jsdelivr.net":     {},
	"unpkg.com":            {},
	// SVG and XLink namespace identifiers, never fetched.
	"www.w3.org": {},
}

type forbiddenPagePattern struct {
	pattern *regexp.Regexp
	message string
}

var forbiddenPagePatterns = []forbiddenPagePattern{
	{regexp.MustCompile(`(?i)<\s*(iframe|frame|frameset|object|embed|form|base|portal|applet)\b`), "must not contain <%s> elements"},
	{regexp.MustCompile(`(?i)http-equiv`), "must not use <meta http-equiv> (the renderer supplies the security policy)"},
	{regexp.MustCompile(`(?i)\bfetch\s*\(`), "must not call fetch(); everything must be inline"},
	{regexp.MustCompile(`(?i)XMLHttpRequest|WebSocket|EventSource|sendBeacon|RTCPeerConnection`), "must not open network connections"},
	{regexp.MustCompile(`(?i)document\s*\.\s*cookie|localStorage|sessionStorage|indexedDB|serviceWorker|caches\s*\.`), "must not use cookies, storage, or service workers"},
	{regexp.MustCompile(`\beval\s*\(|new\s+Function\s*\(|\bimport\s*\(`), "must not use eval, new Function, or dynamic import"},
	{regexp.MustCompile(`(?i)\b(?:window\s*\.\s*)?(?:top|parent|opener)\s*\.\s*(?:location|document|postMessage|frames)`), "must not access the parent window"},
	{regexp.MustCompile(`(?i)window\s*\.\s*open\s*\(|\.postMessage\s*\(`), "must not open windows or post messages"},
	{regexp.MustCompile(`(?i)javascript\s*:`), "must not use javascript: URLs"},
	{regexp.MustCompile(`(?i)\bsrcdoc\b`), "must not use srcdoc"},
	{regexp.MustCompile(`(?i)(?:src|href|action)\s*=\s*["']?//`), "must not use protocol-relative URLs"},
	{regexp.MustCompile(`(?i)<\s*a\b[^>]*\btarget\s*=`), "links must not set target"},
}

// maxTextOnlySpreads allows the contact sheet, whose photos are added by the
// runtime, plus a spare for a deliberate pause.
const maxTextOnlySpreads = 2

var (
	pageSectionPattern  = regexp.MustCompile(`(?is)<section\b.*?</section\s*>`)
	pageRuntimeAlbum    = regexp.MustCompile(`(?i)<main\b[^>]*class\s*=\s*["'][^"']*\bmv-album\b`)
	pageContactSheet    = regexp.MustCompile(`(?i)data-layout\s*=\s*["']contact-sheet["']`)
	pagePhotoGeometry   = regexp.MustCompile(`(?i)\.mv-photo\b[^{}]*\{(?:[^}]*[\s;])?(?:width|height|left|top|position|transform)\s*:`)
	pageFractionalRows  = regexp.MustCompile(`(?i)grid-template-rows\s*:[^;}]*\dfr`)
	pageFixedViewport   = regexp.MustCompile(`(?i)(?:^|[\s;{])height\s*:\s*100[sdl]?vh`)
	pageInvisibleBlocks = regexp.MustCompile(`(?is)<(script|style|noscript|svg|template)\b.*?</(script|style|noscript|svg|template)\s*>|<!--.*?-->|<head\b.*?</head\s*>`)
	pageTagPattern      = regexp.MustCompile(`(?s)<[^>]*>`)
	pageEntityPattern   = regexp.MustCompile(`&[#a-zA-Z0-9]+;`)
	pageURLPattern      = regexp.MustCompile(`(?i)https?://[^\s"'()<>\\]+`)
	pagePhotoPattern    = regexp.MustCompile(`mv-photo://([A-Za-z0-9._-]*)`)
	pageCentredOverlay  = regexp.MustCompile(`(?i)data-place\s*=\s*["']overlay-center["']`)
	pageImgSrcPattern   = regexp.MustCompile(`(?i)<img\b[^>]*?\ssrc\s*=\s*["']([^"']*)["']`)
)

// textOnlySpreads counts <section> blocks without any photo placeholder.
// Sections are rarely nested in generated albums, so a non-greedy match is a
// close enough approximation.
func textOnlySpreads(html string) int {
	count := 0
	for _, block := range pageSectionPattern.FindAllString(html, -1) {
		if !strings.Contains(block, albumPhotoScheme) {
			count++
		}
	}
	return count
}

// centredTitlesOverPeople returns the photos whose spread centres a title
// over their people, so the model can move it to the photo's free area.
func centredTitlesOverPeople(html string, subjects map[string]photoSubject) []string {
	var ids []string
	for _, block := range pageSectionPattern.FindAllString(html, -1) {
		if !pageCentredOverlay.MatchString(block) {
			continue
		}
		for _, match := range pagePhotoPattern.FindAllStringSubmatch(block, -1) {
			if subject, ok := subjects[match[1]]; ok && subject.coversSubject() {
				ids = append(ids, match[1])
				break
			}
		}
	}
	return ids
}

// checkAlbumPage returns every problem it finds, worded as instructions the
// model can act on during repair. An empty slice means the page may be stored.
func checkAlbumPage(html string, rules albumPageRules) []string {
	var issues []string
	add := func(format string, args ...any) { issues = append(issues, fmt.Sprintf(format, args...)) }

	if strings.TrimSpace(html) == "" {
		return []string{"the html field is empty"}
	}
	if len(html) > maxAlbumPageBytes {
		add("the document is %d bytes; keep it under %d bytes by reusing CSS classes", len(html), maxAlbumPageBytes)
	}
	lower := strings.ToLower(html)
	if !strings.HasPrefix(strings.TrimSpace(lower), "<!doctype html") {
		add("the document must start with <!doctype html>")
	}
	for _, tag := range []string{"<html", "<head", "</head>", "<body", "</body>", "</html>"} {
		if !strings.Contains(lower, tag) {
			add("the document is missing %s", tag)
		}
	}
	if !strings.Contains(lower, `name="viewport"`) && !strings.Contains(lower, `name=viewport`) {
		add(`add <meta name="viewport" content="width=device-width, initial-scale=1"> for phones`)
	}
	if !pageRuntimeAlbum.MatchString(html) {
		add(`build the album with the runtime markup: <main class="mv-album"> containing <section class="mv-spread"> pages`)
	}
	if !strings.Contains(html, "mv-spread") || !strings.Contains(html, "mv-photo") {
		add(`every page must be a <section class="mv-spread"> and every photograph a <figure class="mv-photo"><img src="mv-photo://…"></figure>`)
	}
	if pagePhotoGeometry.MatchString(html) {
		add("do not set width, height, left, top, position or transform on .mv-photo or its img; the runtime sizes and places photographs")
	}
	if textOnly := textOnlySpreads(html); textOnly > maxTextOnlySpreads {
		add("%d <section> spreads contain no photograph; give chapter titles a place on their first photo spread instead of a page of their own (only the contact sheet may be text-only)", textOnly)
	}
	for _, id := range centredTitlesOverPeople(html, rules.Subjects) {
		place := "overlay-bottom-left"
		if rules.Subjects[id].TextArea == "top" {
			place = "overlay-top-left"
		}
		add("the centred title on the spread with photo %s covers the people in it; use data-place=%q (the photo's free area is %s)", id, place, firstNonEmpty(rules.Subjects[id].TextArea, "unknown"))
	}
	if pageFractionalRows.MatchString(html) {
		add("do not size grid rows with fr units (grid-template-rows: … 1fr); the runtime lays out photos")
	}
	if pageFixedViewport.MatchString(html) {
		add("do not set height: 100vh/svh/dvh; the runtime owns the page stage")
	}

	for _, rule := range forbiddenPagePatterns {
		if match := rule.pattern.FindStringSubmatch(html); match != nil {
			if strings.Contains(rule.message, "%s") && len(match) > 1 {
				add(rule.message, strings.ToLower(match[1]))
			} else {
				add("%s", rule.message)
			}
		}
	}

	disallowed := map[string]struct{}{}
	for _, raw := range pageURLPattern.FindAllString(html, -1) {
		parsed, err := url.Parse(raw)
		if err != nil {
			disallowed[raw] = struct{}{}
			continue
		}
		if _, ok := allowedPageHosts[strings.ToLower(parsed.Hostname())]; !ok {
			disallowed[parsed.Hostname()] = struct{}{}
		}
	}
	if len(disallowed) > 0 {
		add("remove references to disallowed hosts %s; only fonts.googleapis.com, fonts.gstatic.com, cdnjs.cloudflare.com, cdn.jsdelivr.net and unpkg.com are reachable", joinSortedLimited(disallowed))
	}

	for _, match := range pageImgSrcPattern.FindAllStringSubmatch(html, -1) {
		src := strings.TrimSpace(match[1])
		if !strings.HasPrefix(src, albumPhotoScheme) && !strings.HasPrefix(strings.ToLower(src), "data:") {
			add("<img src=%q> is not a photograph placeholder; use %s<photoId>", truncateRunes(src, 60), albumPhotoScheme)
			break
		}
	}

	featured := map[string]struct{}{}
	unknown := map[string]struct{}{}
	dynamic := false
	for _, match := range pagePhotoPattern.FindAllStringSubmatch(html, -1) {
		id := match[1]
		if id == "" {
			dynamic = true
			continue
		}
		if _, ok := rules.KnownPhotoIDs[id]; ok {
			featured[id] = struct{}{}
		} else {
			unknown[id] = struct{}{}
		}
	}
	if dynamic {
		add("%s must be followed by a literal photo ID; never build photo URLs in JavaScript (e.g. \"%s\" + id) because placeholders are only resolved where they appear literally", albumPhotoScheme, albumPhotoScheme)
	}
	if len(unknown) > 0 {
		add("these photo ids do not exist and must be removed or replaced: %s", joinSortedLimited(unknown))
	}
	var missing []string
	for _, id := range rules.RequiredPhotoIDs {
		if _, ok := featured[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		add("the signature moment photos must appear: %s", strings.Join(missing, ", "))
	}
	if len(featured) < rules.MinFeatured {
		add("only %d distinct photographs are featured; feature at least %d", len(featured), rules.MinFeatured)
	}
	if len(featured) < len(rules.KnownPhotoIDs) && !pageContactSheet.MatchString(html) {
		add(`not every photograph is featured, so end with <section class="mv-spread" data-layout="contact-sheet">`)
	}
	if rules.MaxVisibleWords > 0 {
		if words := visibleWordCount(html); words > rules.MaxVisibleWords {
			add("the album shows %d words of text; keep it under %d. Remove photo descriptions, captions and paragraphs; keep only the names, chapter titles and a few short lines", words, rules.MaxVisibleWords)
		}
	}
	return issues
}

// visibleWordCount approximates the words a reader sees in the static
// markup, ignoring scripts, styles, SVG, comments, and attributes such as
// alt text.
func visibleWordCount(html string) int {
	text := pageInvisibleBlocks.ReplaceAllString(html, " ")
	text = pageTagPattern.ReplaceAllString(text, " ")
	text = pageEntityPattern.ReplaceAllString(text, " ")
	count := 0
	for _, field := range strings.Fields(text) {
		if strings.IndexFunc(field, func(r rune) bool {
			return r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r > 127
		}) >= 0 {
			count++
		}
	}
	return count
}

func joinSortedLimited(values map[string]struct{}) string {
	list := make([]string, 0, len(values))
	for value := range values {
		list = append(list, truncateRunes(value, 60))
	}
	sort.Strings(list)
	if len(list) > 12 {
		list = append(list[:12], fmt.Sprintf("and %d more", len(list)-12))
	}
	return strings.Join(list, ", ")
}
