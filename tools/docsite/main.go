package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
)

//go:embed assets/*
var assets embed.FS

var (
	markdownLinkRE      = regexp.MustCompile(`href="([^"]+)\.md(#[^"]*)?"`)
	firstH1RE           = regexp.MustCompile(`(?s)^\s*<h1[^>]*>.*?</h1>\s*`)
	headingRE           = regexp.MustCompile(`(?s)<h([23]) id="([^"]+)">(.*?)</h[23]>`)
	tagRE               = regexp.MustCompile(`<[^>]+>`)
	markdownH1RE        = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	dayFileRE           = regexp.MustCompile(`day(\d+)`)
	outputRefRE         = regexp.MustCompile(`(?:href|src)="([^"]+)"`)
	outputIDRE          = regexp.MustCompile(`\sid="([^"]+)"`)
	legacyProjectLinkRE = regexp.MustCompile(`https://github\.com/geektutu/7days-golang|https://github\.com/glaxry/7days-golang/(tree|blob)/master|https://geektutu\.com/post/(gee(-day[1-7])?|geecache(-day[1-7])?|geeorm(-day[1-7])?|geerpc(-day[1-7])?|7days-golang-q1|quick-go-wasm)\.html|ghttps://github\.com/`)
)

type document struct {
	SourcePath string
	OutputPath string
	Title      string
	NavTitle   string
	Group      string
	Body       string
	Meta       map[string]string
}

type navGroup struct {
	Name  string
	Items []navItem
}

type navItem struct {
	Title   string
	Href    string
	Current bool
	Search  string
}

type tocItem struct {
	Level int
	ID    string
	Title string
}

type pageData struct {
	Title        string
	Description  string
	Group        string
	Content      template.HTML
	HomeHero     template.HTML
	Nav          []navGroup
	TOC          []tocItem
	StylesHref   string
	ScriptHref   string
	HomeHref     string
	ModernHref   string
	Previous     *navItem
	Next         *navItem
	PagePosition string
}

type searchRecord struct {
	Title string `json:"title"`
	Group string `json:"group"`
	URL   string `json:"url"`
}

func main() {
	rootFlag := flag.String("root", "../..", "repository root")
	outFlag := flag.String("out", "docs", "output directory relative to repository root")
	check := flag.Bool("check", false, "verify generated output is current")
	flag.Parse()

	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		fatal(err)
	}
	out := filepath.Join(root, filepath.FromSlash(*outFlag))
	if err := ensureOutputIsSafe(root, out); err != nil {
		fatal(err)
	}

	if *check {
		temp, err := os.MkdirTemp("", "7days-golang-docsite-")
		if err != nil {
			fatal(err)
		}
		defer os.RemoveAll(temp)
		if err := buildSite(root, temp); err != nil {
			fatal(err)
		}
		if err := compareTrees(temp, out); err != nil {
			fatal(fmt.Errorf("HTML documentation is out of date: %w", err))
		}
		fmt.Println("HTML documentation is up to date.")
		return
	}

	if err := os.RemoveAll(out); err != nil {
		fatal(err)
	}
	if err := buildSite(root, out); err != nil {
		fatal(err)
	}
	fmt.Printf("Generated HTML documentation in %s.\n", out)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func ensureOutputIsSafe(root, out string) error {
	rel, err := filepath.Rel(root, out)
	if err != nil {
		return err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return fmt.Errorf("refusing output outside repository: %s", out)
	}
	return nil
}

func buildSite(root, out string) error {
	docs, err := loadDocuments(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "assets"), 0o755); err != nil {
		return err
	}
	for _, name := range []string{"styles.css", "app.js"} {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, "assets", name), data, 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(out, ".nojekyll"), nil, 0o644); err != nil {
		return err
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(rendererhtml.WithUnsafe()),
	)

	for i := range docs {
		if err := renderDocument(root, out, markdown, docs, i); err != nil {
			return fmt.Errorf("render %s: %w", docs[i].SourcePath, err)
		}
	}
	if err := copyDocumentAssets(root, out); err != nil {
		return err
	}
	if err := writeSearchIndex(out, docs); err != nil {
		return err
	}
	return validateOutput(out)
}

func loadDocuments(root string) ([]document, error) {
	paths := []string{"README.md", "MODERNIZATION.md", "gee-web/README.md", "demo-wasm/README.md"}
	for _, dir := range []string{"gee-web/doc", "gee-cache/doc", "gee-orm/doc", "gee-rpc/doc", "questions"} {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	paths = slices.Compact(paths)
	docs := make([]document, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		if err := validateSourceLinks(path, string(data)); err != nil {
			return nil, err
		}
		body, meta := splitFrontMatter(string(data))
		title := documentTitle(path, body, meta)
		docs = append(docs, document{
			SourcePath: path,
			OutputPath: outputPath(path),
			Title:      title,
			NavTitle:   navigationTitle(path, title, meta),
			Group:      documentGroup(path),
			Body:       body,
			Meta:       meta,
		})
	}

	sort.SliceStable(docs, func(i, j int) bool {
		return documentOrder(docs[i].SourcePath) < documentOrder(docs[j].SourcePath)
	})
	return docs, nil
}

func validateSourceLinks(path, source string) error {
	if legacy := legacyProjectLinkRE.FindString(source); legacy != "" {
		return fmt.Errorf("%s: legacy or malformed project link %q", path, legacy)
	}
	return nil
}

func splitFrontMatter(source string) (string, map[string]string) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	meta := make(map[string]string)
	if !strings.HasPrefix(source, "---\n") {
		return source, meta
	}
	end := strings.Index(source[4:], "\n---\n")
	if end < 0 {
		return source, meta
	}
	front := source[4 : 4+end]
	for line := range strings.SplitSeq(front, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value != "" {
			meta[strings.TrimSpace(key)] = value
		}
	}
	return source[4+end+5:], meta
}

func documentTitle(path, body string, meta map[string]string) string {
	if path == "README.md" {
		return "7天用 Go 从零实现系列"
	}
	if path == "MODERNIZATION.md" {
		return "现代 Go 迁移说明"
	}
	if title := strings.TrimSpace(meta["title"]); title != "" {
		return title
	}
	if match := markdownH1RE.FindStringSubmatch(body); len(match) == 2 {
		return plainText(match[1])
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.ReplaceAll(name, "-", " ")
}

func navigationTitle(path, title string, meta map[string]string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch path {
	case "README.md":
		return "首页"
	case "MODERNIZATION.md":
		return "现代化迁移"
	case "demo-wasm/README.md":
		return "WebAssembly 示例"
	case "gee-web/README.md":
		return "项目 README"
	}
	if base == "gee" || base == "geecache" || base == "geeorm" || base == "geerpc" {
		return "系列总览"
	}
	if bookTitle := strings.TrimSpace(meta["book_title"]); bookTitle != "" {
		return bookTitle
	}
	return title
}

func documentGroup(path string) string {
	switch {
	case path == "README.md" || path == "MODERNIZATION.md":
		return "开始"
	case strings.HasPrefix(path, "gee-web/"):
		return "Gee Web"
	case strings.HasPrefix(path, "gee-cache/"):
		return "GeeCache"
	case strings.HasPrefix(path, "gee-orm/"):
		return "GeeORM"
	case strings.HasPrefix(path, "gee-rpc/"):
		return "GeeRPC"
	default:
		return "其他"
	}
}

func documentOrder(path string) int {
	group := map[string]int{
		"开始": 0, "Gee Web": 100, "GeeCache": 200, "GeeORM": 300, "GeeRPC": 400, "其他": 500,
	}[documentGroup(path)]
	if path == "README.md" {
		return group
	}
	if path == "MODERNIZATION.md" {
		return group + 1
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if base == "gee" || base == "geecache" || base == "geeorm" || base == "geerpc" {
		return group
	}
	if match := dayFileRE.FindStringSubmatch(base); len(match) == 2 {
		day, _ := strconv.Atoi(match[1])
		return group + day
	}
	return group + 50
}

func outputPath(source string) string {
	if source == "README.md" {
		return "index.html"
	}
	return strings.TrimSuffix(source, filepath.Ext(source)) + ".html"
}

func renderDocument(root, out string, markdown goldmark.Markdown, docs []document, index int) error {
	doc := docs[index]
	var converted bytes.Buffer
	if err := markdown.Convert([]byte(doc.Body), &converted); err != nil {
		return err
	}
	content := rewriteMarkdownLinks(converted.String())
	if doc.SourcePath == "README.md" {
		content = strings.ReplaceAll(content, `href="docs/index.html"`, `href="index.html"`)
	}
	content = firstH1RE.ReplaceAllString(content, "")

	pageDir := filepath.Dir(filepath.Join(out, filepath.FromSlash(doc.OutputPath)))
	stylesHref := relativeURL(pageDir, filepath.Join(out, "assets", "styles.css"))
	scriptHref := relativeURL(pageDir, filepath.Join(out, "assets", "app.js"))
	homeHref := relativeURL(pageDir, filepath.Join(out, "index.html"))
	modernHref := relativeURL(pageDir, filepath.Join(out, "MODERNIZATION.html"))

	data := pageData{
		Title:        doc.Title,
		Description:  pageDescription(doc),
		Group:        doc.Group,
		Content:      template.HTML(content),
		Nav:          buildNavigation(out, pageDir, docs, doc.OutputPath),
		TOC:          extractTOC(content),
		StylesHref:   stylesHref,
		ScriptHref:   scriptHref,
		HomeHref:     homeHref,
		ModernHref:   modernHref,
		PagePosition: fmt.Sprintf("%d / %d", index+1, len(docs)),
	}
	if doc.SourcePath == "README.md" {
		data.HomeHero = template.HTML(homeHero(modernHref))
	}
	if index > 0 {
		item := navItem{Title: docs[index-1].NavTitle, Href: relativeURL(pageDir, filepath.Join(out, filepath.FromSlash(docs[index-1].OutputPath)))}
		data.Previous = &item
	}
	if index+1 < len(docs) {
		item := navItem{Title: docs[index+1].NavTitle, Href: relativeURL(pageDir, filepath.Join(out, filepath.FromSlash(docs[index+1].OutputPath)))}
		data.Next = &item
	}

	destination := filepath.Join(out, filepath.FromSlash(doc.OutputPath))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	var page bytes.Buffer
	if err := pageTemplate.Execute(&page, data); err != nil {
		return err
	}
	return os.WriteFile(destination, trimTrailingWhitespace(page.Bytes()), 0o644)
}

func trimTrailingWhitespace(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return []byte(strings.Join(lines, "\n"))
}

func rewriteMarkdownLinks(content string) string {
	return markdownLinkRE.ReplaceAllString(content, `href="$1.html$2"`)
}

func extractTOC(content string) []tocItem {
	matches := headingRE.FindAllStringSubmatch(content, -1)
	items := make([]tocItem, 0, len(matches))
	for _, match := range matches {
		level, _ := strconv.Atoi(match[1])
		title := strings.TrimSpace(html.UnescapeString(tagRE.ReplaceAllString(match[3], "")))
		if title == "" {
			continue
		}
		items = append(items, tocItem{Level: level, ID: html.UnescapeString(match[2]), Title: title})
	}
	return items
}

func buildNavigation(out, pageDir string, docs []document, current string) []navGroup {
	order := []string{"开始", "Gee Web", "GeeCache", "GeeORM", "GeeRPC", "其他"}
	groups := make(map[string][]navItem)
	for _, doc := range docs {
		groups[doc.Group] = append(groups[doc.Group], navItem{
			Title:   doc.NavTitle,
			Href:    relativeURL(pageDir, filepath.Join(out, filepath.FromSlash(doc.OutputPath))),
			Current: doc.OutputPath == current,
			Search:  strings.ToLower(doc.Title + " " + doc.NavTitle + " " + doc.Group),
		})
	}
	result := make([]navGroup, 0, len(order))
	for _, name := range order {
		if len(groups[name]) > 0 {
			result = append(result, navGroup{Name: name, Items: groups[name]})
		}
	}
	return result
}

func relativeURL(fromDir, target string) string {
	rel, err := filepath.Rel(fromDir, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

func pageDescription(doc document) string {
	description := strings.TrimSpace(doc.Meta["description"])
	if description == "" {
		description = doc.Title + "，基于 Go 1.26 更新的七天从零实现教程。"
	}
	return truncateRunes(description, 170)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}

func plainText(value string) string {
	value = strings.NewReplacer("**", "", "__", "", "`", "", "*", "", "_", "").Replace(value)
	return strings.TrimSpace(value)
}

func homeHero(modernHref string) string {
	return `<section class="hero" aria-labelledby="hero-title">
  <div class="hero-orbit hero-orbit-one"></div>
  <div class="hero-orbit hero-orbit-two"></div>
  <div class="hero-content">
    <span class="hero-kicker">GO 1.26 · 2026 现代化版本</span>
    <h1 id="hero-title">七天用 Go，<br><span>从原理到实现</span></h1>
    <p>四套从零实现教程，保留循序渐进的 Day 结构，并同步到当前语言、标准库与依赖版本。</p>
    <div class="hero-actions">
	    <a class="button button-primary" href="#article-start">开始阅读</a>
      <a class="button button-secondary" href="` + template.HTMLEscapeString(modernHref) + `">查看迁移说明</a>
    </div>
  </div>
  <dl class="hero-stats" aria-label="教程规模">
    <div><dt>37</dt><dd>篇教程</dd></div>
    <div><dt>45</dt><dd>个 Go 模块</dd></div>
    <div><dt>4</dt><dd>套核心项目</dd></div>
  </dl>
</section>`
}

func copyDocumentAssets(root, out string) error {
	for _, dir := range []string{"gee-web/doc", "gee-cache/doc", "gee-orm/doc", "gee-rpc/doc", "questions"} {
		sourceDir := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || strings.ToLower(filepath.Ext(path)) == ".md" {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if !slices.Contains([]string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"}, ext) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			destination := filepath.Join(out, rel)
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(destination, data, 0o644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSearchIndex(out string, docs []document) error {
	records := make([]searchRecord, 0, len(docs))
	for _, doc := range docs {
		records = append(records, searchRecord{Title: doc.NavTitle, Group: doc.Group, URL: doc.OutputPath})
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(out, "search-index.json"), data, 0o644)
}

func validateOutput(root string) error {
	var problems []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".html" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		if !strings.HasPrefix(content, "<!doctype html>") {
			problems = append(problems, filepath.ToSlash(path)+": missing doctype")
		}
		ids := make(map[string]bool)
		for _, match := range outputIDRE.FindAllStringSubmatch(content, -1) {
			ids[html.UnescapeString(match[1])] = true
		}
		for _, match := range outputRefRE.FindAllStringSubmatch(content, -1) {
			ref := html.UnescapeString(match[1])
			parsed, err := url.Parse(ref)
			if err != nil || parsed.Scheme != "" || strings.HasPrefix(ref, "//") {
				continue
			}
			if parsed.Path == "" {
				if parsed.Fragment != "" {
					fragment, _ := url.PathUnescape(parsed.Fragment)
					if !ids[fragment] {
						problems = append(problems, filepath.ToSlash(path)+": missing anchor #"+fragment)
					}
				}
				continue
			}
			ext := strings.ToLower(filepath.Ext(parsed.Path))
			if ext != ".html" && ext != ".css" && ext != ".js" && !slices.Contains([]string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"}, ext) {
				continue
			}
			target := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(parsed.Path)))
			if _, err := os.Stat(target); err != nil {
				rel, _ := filepath.Rel(root, path)
				problems = append(problems, filepath.ToSlash(rel)+": missing "+ref)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	if len(problems) > 20 {
		problems = append(problems[:20], "…")
	}
	return errors.New("generated site validation failed: " + strings.Join(problems, ", "))
}

func compareTrees(wantRoot, gotRoot string) error {
	want, err := treeContents(wantRoot)
	if err != nil {
		return err
	}
	got, err := treeContents(gotRoot)
	if err != nil {
		return err
	}
	var differences []string
	for path, wantData := range want {
		gotData, ok := got[path]
		if !ok {
			differences = append(differences, "missing "+path)
			continue
		}
		if !bytes.Equal(wantData, gotData) {
			differences = append(differences, "changed "+path)
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			differences = append(differences, "unexpected "+path)
		}
	}
	if len(differences) == 0 {
		return nil
	}
	sort.Strings(differences)
	if len(differences) > 12 {
		differences = append(differences[:12], "…")
	}
	return errors.New(strings.Join(differences, ", "))
}

func treeContents(root string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(rel)] = data
		return nil
	})
	return result, err
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="{{.Description}}">
  <meta name="color-scheme" content="light dark">
  <title>{{.Title}} · 7天用 Go</title>
  <link rel="stylesheet" href="{{.StylesHref}}">
  <script defer src="{{.ScriptHref}}"></script>
</head>
<body>
  <div class="reading-progress" aria-hidden="true"><span></span></div>
  <a class="skip-link" href="#main-content">跳到正文</a>
  <header class="topbar">
    <div class="topbar-inner">
      <button class="icon-button menu-button" type="button" aria-label="打开教程导航" aria-controls="sidebar" aria-expanded="false"><span></span><span></span><span></span></button>
      <a class="brand" href="{{.HomeHref}}" aria-label="7天用 Go 首页"><span class="brand-mark">7</span><span><strong>7days</strong><small>golang</small></span></a>
      <nav class="top-links" aria-label="顶部导航">
        <a href="{{.HomeHref}}">教程首页</a>
        <a href="{{.ModernHref}}">迁移说明</a>
        <span class="version-pill">Go 1.26</span>
      </nav>
      <button class="icon-button theme-button" type="button" aria-label="切换深浅色主题"><span class="theme-icon" aria-hidden="true">◐</span></button>
    </div>
  </header>
  <div class="sidebar-overlay" aria-hidden="true"></div>
  <div class="doc-shell">
    <aside class="sidebar" id="sidebar" aria-label="教程目录">
      <div class="sidebar-search"><label for="nav-search">查找教程</label><div><span aria-hidden="true">⌕</span><input id="nav-search" type="search" placeholder="标题或系列…" autocomplete="off"></div></div>
      <nav class="sidebar-nav">
        {{range .Nav}}<section class="nav-group"><h2>{{.Name}}</h2>{{range .Items}}<a class="nav-item{{if .Current}} is-current{{end}}" href="{{.Href}}" data-search="{{.Search}}"{{if .Current}} aria-current="page"{{end}}>{{.Title}}</a>{{end}}</section>{{end}}
        <p class="search-empty" hidden>没有匹配的教程</p>
      </nav>
    </aside>
    <main id="main-content" class="main-column">
      {{.HomeHero}}
      <article class="article-card" id="article-start">
        <header class="article-header">
          <div class="article-kicker"><span>{{.Group}}</span><span>{{.PagePosition}}</span></div>
          <h1>{{.Title}}</h1>
          <p>{{.Description}}</p>
        </header>
        <div class="prose">{{.Content}}</div>
        <nav class="page-nav" aria-label="上一篇和下一篇">
          {{if .Previous}}<a class="page-nav-item previous" href="{{.Previous.Href}}"><span>上一篇</span><strong>{{.Previous.Title}}</strong></a>{{else}}<span></span>{{end}}
          {{if .Next}}<a class="page-nav-item next" href="{{.Next.Href}}"><span>下一篇</span><strong>{{.Next.Title}}</strong></a>{{end}}
        </nav>
      </article>
      <footer class="site-footer"><span>基于 Go 1.26 更新</span><span>Markdown 与 HTML 同源生成</span></footer>
    </main>
    <aside class="toc" aria-label="本页目录">
      {{if .TOC}}<h2>本页目录</h2><nav>{{range .TOC}}<a class="toc-level-{{.Level}}" href="#{{.ID}}">{{.Title}}</a>{{end}}</nav>{{end}}
    </aside>
  </div>
</body>
</html>
`))
