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

func TestValidateSourceLinks(t *testing.T) {
	tests := []struct {
		name    string
		link    string
		wantErr bool
	}{
		{name: "current repository", link: "https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee.md"},
		{name: "external reference", link: "https://geektutu.com/post/quick-golang.html"},
		{name: "old repository", link: "https://github.com/geektutu/7days-golang", wantErr: true},
		{name: "old tutorial", link: "https://geektutu.com/post/geecache-day2.html", wantErr: true},
		{name: "old branch", link: "https://github.com/glaxry/7days-golang/tree/master/gee-rpc", wantErr: true},
		{name: "malformed URL", link: "ghttps://github.com/glaxry/7days-golang/tree/main/gee-rpc", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSourceLinks("test.md", test.link)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateSourceLinks() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
