package service

import (
	"strings"
	"testing"
)

const validAlbumPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1"><title>Julian &amp; Sofia</title>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Fraunces">
<style>.mv-text h2{font-size:40px}.mv-photo img{filter:saturate(1.1)}</style></head>
<body><svg xmlns="http://www.w3.org/2000/svg"></svg>
<main class="mv-album" data-transition="turn">
<section class="mv-spread" data-layout="hero"><figure class="mv-photo" data-fill="bleed" data-focal="0.5 0.4"><img src="mv-photo://a1" alt=""></figure>
<div class="mv-text" data-place="overlay-bottom-left"><h1>Julian &amp; Sofia</h1></div></section>
<section class="mv-spread" data-layout="duo"><figure class="mv-photo"><img src="mv-photo://a2" alt=""></figure></section>
<section class="mv-spread" data-layout="contact-sheet"></section>
</main></body></html>`

func pageRules() albumPageRules {
	return albumPageRules{
		KnownPhotoIDs:    map[string]struct{}{"a1": {}, "a2": {}, "a3": {}},
		RequiredPhotoIDs: []string{"a1"},
		MinFeatured:      2,
	}
}

func TestCheckAlbumPageAcceptsRuntimeAlbum(t *testing.T) {
	if issues := checkAlbumPage(validAlbumPage, pageRules()); len(issues) != 0 {
		t.Fatalf("expected a valid album, got %v", issues)
	}
}

func TestCheckAlbumPageRejectsUnsafeOrIncompletePages(t *testing.T) {
	cases := map[string]struct {
		html string
		want string
	}{
		"network":          {strings.Replace(validAlbumPage, "</main>", "</main><script>fetch('/x')</script>", 1), "fetch"},
		"storage":          {strings.Replace(validAlbumPage, "</main>", "</main><script>localStorage.x=1</script>", 1), "storage"},
		"parent":           {strings.Replace(validAlbumPage, "</main>", "</main><script>window.parent.postMessage(1,'*')</script>", 1), "parent window"},
		"external image":   {strings.Replace(validAlbumPage, "mv-photo://a1", "https://evil.example/x.jpg", 1), "evil.example"},
		"unknown photo":    {strings.Replace(validAlbumPage, "mv-photo://a2", "mv-photo://zz", 1), "zz"},
		"signature":        {strings.Replace(validAlbumPage, "mv-photo://a1", "mv-photo://a3", 1), "signature moment"},
		"iframe":           {strings.Replace(validAlbumPage, "<main", "<iframe></iframe><main", 1), "<iframe>"},
		"csp override":     {strings.Replace(validAlbumPage, "<title>", `<meta http-equiv="refresh" content="0"><title>`, 1), "http-equiv"},
		"doctype":          {strings.TrimPrefix(validAlbumPage, "<!doctype html>"), "doctype"},
		"fixed height":     {strings.Replace(validAlbumPage, ".mv-text h2{font-size:40px}", ".hero{height:100svh}", 1), "100vh"},
		"fr rows":          {strings.Replace(validAlbumPage, ".mv-text h2{font-size:40px}", ".g{grid-template-rows:repeat(6,1fr)}", 1), "grid rows"},
		"title pages":      {strings.Replace(validAlbumPage, `<section class="mv-spread" data-layout="duo">`, strings.Repeat(`<section class="mv-spread"><h2>Chapter</h2></section>`, 3)+`<section class="mv-spread" data-layout="duo">`, 1), "no photograph"},
		"dynamic photo":    {strings.Replace(validAlbumPage, "</main>", "</main><script>var u='mv-photo://'+id;</script>", 1), "literal photo ID"},
		"no album":         {strings.Replace(validAlbumPage, `<main class="mv-album" data-transition="turn">`, "<main>", 1), "runtime markup"},
		"photo geometry":   {strings.Replace(validAlbumPage, ".mv-text h2{font-size:40px}", ".mv-photo{width:300px}", 1), "runtime sizes"},
		"no contact sheet": {strings.Replace(validAlbumPage, `data-layout="contact-sheet"`, `data-layout="text"`, 1), "contact-sheet"},
	}
	for name, tc := range cases {
		issues := checkAlbumPage(tc.html, pageRules())
		if !strings.Contains(strings.Join(issues, "\n"), tc.want) {
			t.Errorf("%s: expected an issue mentioning %q, got %v", name, tc.want, issues)
		}
	}
}

func TestCheckAlbumPageAllowsNoContactSheetWhenEveryPhotoIsFeatured(t *testing.T) {
	html := strings.Replace(validAlbumPage, `<section class="mv-spread" data-layout="contact-sheet"></section>`,
		`<section class="mv-spread" data-layout="hero"><figure class="mv-photo"><img src="mv-photo://a3" alt=""></figure></section>`, 1)
	if issues := checkAlbumPage(html, pageRules()); len(issues) != 0 {
		t.Fatalf("expected a valid album, got %v", issues)
	}
}

func TestVisibleWordCountIgnoresScriptsStylesAndAltText(t *testing.T) {
	html := `<!doctype html><html><head><title>Ten words in the head do not count at all here</title>
<style>.a{content:"style words do not count"}</style></head><body>
<h1>Julian and Sofia</h1><img src="mv-photo://a1" alt="long alt text describing the couple at the altar">
<script>var s = "script words do not count either";</script><!-- nor comments -->
<p>One &amp; two</p></body></html>`
	if got := visibleWordCount(html); got != 5 {
		t.Fatalf("expected 5 visible words, got %d", got)
	}
	wordy := strings.Replace(validAlbumPage, "<h1>Julian &amp; Sofia</h1>",
		"<p>"+strings.Repeat("A beautiful couple at their wedding. ", 20)+"</p>", 1)
	r := pageRules()
	r.MaxVisibleWords = 50
	if issues := checkAlbumPage(wordy, r); !strings.Contains(strings.Join(issues, "\n"), "words of text") {
		t.Fatalf("expected a word budget issue, got %v", issues)
	}
	if issues := checkAlbumPage(validAlbumPage, r); len(issues) != 0 {
		t.Fatalf("short page must pass the budget, got %v", issues)
	}
}

func TestCheckAlbumPageKeepsCentredTitlesOffPeople(t *testing.T) {
	centred := strings.Replace(validAlbumPage, `data-place="overlay-bottom-left"`, `data-place="overlay-center"`, 1)
	r := pageRules()
	r.Subjects = map[string]photoSubject{"a1": {FocalX: 0.5, FocalY: 0.45, TextArea: "top"}}

	issues := strings.Join(checkAlbumPage(centred, r), "\n")
	if !strings.Contains(issues, `covers the people`) || !strings.Contains(issues, `"overlay-top-left"`) {
		t.Fatalf("expected a centred-title issue pointing to the free area, got %v", issues)
	}

	// People off to one side leave the centre free.
	r.Subjects["a1"] = photoSubject{FocalX: 0.2, FocalY: 0.5, TextArea: "right"}
	if issues := checkAlbumPage(centred, r); len(issues) != 0 {
		t.Fatalf("an off-centre subject should allow a centred title, got %v", issues)
	}
	// The original page keeps its title in the corner.
	r.Subjects["a1"] = photoSubject{FocalX: 0.5, FocalY: 0.45, TextArea: "top"}
	if issues := checkAlbumPage(validAlbumPage, r); len(issues) != 0 {
		t.Fatalf("a corner title is fine, got %v", issues)
	}
}
