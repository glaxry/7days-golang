package main

import "testing"

func TestSplitFrontMatter(t *testing.T) {
	body, meta := splitFrontMatter("---\ntitle: 示例\nbook_title: Day1 入门\n---\n# 正文\n")
	if meta["title"] != "示例" || meta["book_title"] != "Day1 入门" || body != "# 正文\n" {
		t.Fatalf("unexpected result: body=%q meta=%v", body, meta)
	}
}

func TestRewriteMarkdownLinks(t *testing.T) {
	got := rewriteMarkdownLinks(`<a href="README.md">首页</a><a href="guide.md#part">章节</a>`)
	want := `<a href="README.html">首页</a><a href="guide.html#part">章节</a>`
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestEnsureOutputIsSafe(t *testing.T) {
	root := t.TempDir()
	if err := ensureOutputIsSafe(root, root); err == nil {
		t.Fatal("repository root must not be accepted as output")
	}
	if err := ensureOutputIsSafe(root, root+"-other"); err == nil {
		t.Fatal("outside path must not be accepted as output")
	}
}
