package server

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type uiDocSpec struct {
	Key         string
	Filename    string
	Label       string
	Description string
}

type resolvedUIDoc struct {
	spec uiDocSpec
	path string
}

type renderedDocHeading struct {
	Level int
	ID    string
	Text  string
}

var uiDocs = []uiDocSpec{
	{
		Key:         "getting_started",
		Filename:    "GETTING_STARTED.md",
		Label:       "Getting Started",
		Description: "Setup, run, refresh, and troubleshoot the local daemon and tray workflow.",
	},
	{
		Key:         "architecture",
		Filename:    "ARCHITECTURE.md",
		Label:       "Architecture",
		Description: "Entry points, major subsystems, local HTTP surface, persistence, and security boundaries.",
	},
}

func resolveUIDoc(name string) (resolvedUIDoc, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	for _, spec := range uiDocs {
		if spec.Key != key {
			continue
		}
		path, err := resolveUIDocFile(spec.Filename)
		if err != nil {
			return resolvedUIDoc{}, err
		}
		return resolvedUIDoc{spec: spec, path: path}, nil
	}
	return resolvedUIDoc{}, fmt.Errorf("unknown doc")
}

func resolveUIDocFile(filename string) (string, error) {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		candidates = append(candidates, docCandidatesFromRoot(cwd, filename)...)
	}
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		candidates = append(candidates, docCandidatesFromRoot(filepath.Dir(exe), filename)...)
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		info, err := os.Stat(clean)
		if err != nil || info.IsDir() {
			continue
		}
		return clean, nil
	}
	return "", fmt.Errorf("doc is not available on this runtime")
}

func resolveUIAssetFile(pathSuffix string) (string, error) {
	pathSuffix = filepath.Clean(strings.TrimSpace(pathSuffix))
	if pathSuffix == "." || pathSuffix == "" || strings.HasPrefix(pathSuffix, "..") {
		return "", fmt.Errorf("unknown asset")
	}
	return resolveUIDocFile(filepath.Join("assets", pathSuffix))
}

func docCandidatesFromRoot(root string, filename string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}

	out := make([]string, 0, 8)
	for {
		out = append(out, filepath.Join(root, "docs", filename))
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	return out
}

func availableUIDocs() []map[string]any {
	out := make([]map[string]any, 0, len(uiDocs))
	for _, spec := range uiDocs {
		path, err := resolveUIDocFile(spec.Filename)
		if err != nil {
			continue
		}
		out = append(out, map[string]any{
			"key":         spec.Key,
			"name":        spec.Filename,
			"label":       spec.Label,
			"description": spec.Description,
			"path":        path,
		})
	}
	return out
}

func renderMarkdownDocument(markdown string) (string, []map[string]any) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var out strings.Builder
	var paragraph []string
	headings := make([]map[string]any, 0, 12)
	slugCounts := map[string]int{}
	tabGroupCount := 0
	inCodeBlock := false
	inUnorderedList := false
	inOrderedList := false

	closeParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		out.WriteString("<p>")
		out.WriteString(renderInlineMarkdown(strings.Join(paragraph, " ")))
		out.WriteString("</p>\n")
		paragraph = nil
	}
	closeLists := func() {
		if inUnorderedList {
			out.WriteString("</ul>\n")
			inUnorderedList = false
		}
		if inOrderedList {
			out.WriteString("</ol>\n")
			inOrderedList = false
		}
	}

	for index := 0; index < len(lines); index++ {
		rawLine := lines[index]
		trimmed := strings.TrimSpace(rawLine)
		if inCodeBlock {
			if strings.HasPrefix(trimmed, "```") {
				out.WriteString("</code></pre>\n")
				inCodeBlock = false
				continue
			}
			out.WriteString(html.EscapeString(rawLine))
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			closeParagraph()
			closeLists()
			out.WriteString("<pre class=\"doc-code\"><code>")
			inCodeBlock = true
			continue
		}
		if alt, src, ok := parseMarkdownImage(trimmed); ok {
			closeParagraph()
			closeLists()
			out.WriteString("<figure class=\"doc-image\">")
			out.WriteString("<img src=\"")
			out.WriteString(html.EscapeString(src))
			out.WriteString("\" alt=\"")
			out.WriteString(html.EscapeString(alt))
			out.WriteString("\" loading=\"lazy\" />")
			if strings.TrimSpace(alt) != "" {
				out.WriteString("<figcaption>")
				out.WriteString(html.EscapeString(alt))
				out.WriteString("</figcaption>")
			}
			out.WriteString("</figure>\n")
			continue
		}
		if trimmed == ":::tabs" {
			closeParagraph()
			closeLists()
			tabLines := make([]string, 0, 16)
			for index+1 < len(lines) {
				index++
				nextLine := lines[index]
				if strings.TrimSpace(nextLine) == ":::" {
					break
				}
				tabLines = append(tabLines, nextLine)
			}
			tabGroupCount++
			out.WriteString(renderMarkdownTabsBlock(tabLines, tabGroupCount))
			continue
		}
		if trimmed == "" {
			closeParagraph()
			closeLists()
			continue
		}
		if level, text, ok := parseMarkdownHeading(trimmed); ok {
			closeParagraph()
			closeLists()
			id := uniqueHeadingID(text, slugCounts)
			headings = append(headings, map[string]any{
				"level": level,
				"id":    id,
				"text":  text,
			})
			fmt.Fprintf(&out, "<h%d id=\"%s\">%s</h%d>\n", level, id, renderInlineMarkdown(text), level)
			continue
		}
		if item, ok := parseMarkdownListItem(trimmed); ok {
			closeParagraph()
			if inOrderedList {
				out.WriteString("</ol>\n")
				inOrderedList = false
			}
			if !inUnorderedList {
				out.WriteString("<ul>\n")
				inUnorderedList = true
			}
			out.WriteString("<li>")
			out.WriteString(renderInlineMarkdown(item))
			out.WriteString("</li>\n")
			continue
		}
		if item, ok := parseMarkdownOrderedItem(trimmed); ok {
			closeParagraph()
			if inUnorderedList {
				out.WriteString("</ul>\n")
				inUnorderedList = false
			}
			if !inOrderedList {
				out.WriteString("<ol>\n")
				inOrderedList = true
			}
			out.WriteString("<li>")
			out.WriteString(renderInlineMarkdown(item))
			out.WriteString("</li>\n")
			continue
		}
		closeLists()
		paragraph = append(paragraph, trimmed)
	}

	closeParagraph()
	closeLists()
	if inCodeBlock {
		out.WriteString("</code></pre>\n")
	}
	return out.String(), headings
}

func renderMarkdownTabsBlock(lines []string, groupNumber int) string {
	type markdownTab struct {
		Label string
		Body  string
	}

	tabs := make([]markdownTab, 0, 6)
	currentLabel := ""
	currentLines := make([]string, 0, len(lines))
	flushTab := func() {
		if strings.TrimSpace(currentLabel) == "" {
			return
		}
		tabs = append(tabs, markdownTab{
			Label: currentLabel,
			Body:  strings.Join(currentLines, "\n"),
		})
		currentLines = currentLines[:0]
	}

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "@tab ") {
			flushTab()
			currentLabel = strings.TrimSpace(strings.TrimPrefix(trimmed, "@tab "))
			continue
		}
		currentLines = append(currentLines, rawLine)
	}
	flushTab()
	if len(tabs) == 0 {
		return ""
	}

	var out strings.Builder
	fmt.Fprintf(&out, "<div class=\"doc-tabs\" data-doc-tabs=\"group-%d\">\n", groupNumber)
	out.WriteString("<div class=\"doc-tab-list\" role=\"tablist\">\n")
	for idx, tab := range tabs {
		tabID := fmt.Sprintf("group-%d-tab-%d", groupNumber, idx)
		panelID := fmt.Sprintf("group-%d-panel-%d", groupNumber, idx)
		activeClass := ""
		selected := "false"
		tabIndex := "-1"
		if idx == 0 {
			activeClass = " active"
			selected = "true"
			tabIndex = "0"
		}
		fmt.Fprintf(&out, "<button type=\"button\" class=\"doc-tab%s\" role=\"tab\" aria-selected=\"%s\" tabindex=\"%s\" id=\"%s\" aria-controls=\"%s\" data-doc-tab=\"%s\">%s</button>\n",
			activeClass, selected, tabIndex, tabID, panelID, panelID, html.EscapeString(tab.Label))
	}
	out.WriteString("</div>\n")
	out.WriteString("<div class=\"doc-tab-panels\">\n")
	for idx, tab := range tabs {
		panelID := fmt.Sprintf("group-%d-panel-%d", groupNumber, idx)
		tabID := fmt.Sprintf("group-%d-tab-%d", groupNumber, idx)
		activeClass := ""
		hidden := " hidden"
		if idx == 0 {
			activeClass = " active"
			hidden = ""
		}
		bodyHTML, _ := renderMarkdownDocument(tab.Body)
		fmt.Fprintf(&out, "<section class=\"doc-tab-panel%s\" role=\"tabpanel\" id=\"%s\" aria-labelledby=\"%s\"%s>%s</section>\n",
			activeClass, panelID, tabID, hidden, bodyHTML)
	}
	out.WriteString("</div>\n")
	out.WriteString("</div>\n")
	return out.String()
}

func parseMarkdownHeading(line string) (int, string, bool) {
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level+1:]), true
}

func parseMarkdownListItem(line string) (string, bool) {
	if !strings.HasPrefix(line, "- ") {
		return "", false
	}
	return strings.TrimSpace(line[2:]), true
}

func parseMarkdownOrderedItem(line string) (string, bool) {
	index := 0
	for index < len(line) && line[index] >= '0' && line[index] <= '9' {
		index++
	}
	if index == 0 || index+1 >= len(line) || line[index] != '.' || line[index+1] != ' ' {
		return "", false
	}
	return strings.TrimSpace(line[index+2:]), true
}

func parseMarkdownImage(line string) (string, string, bool) {
	if !strings.HasPrefix(line, "![") {
		return "", "", false
	}
	mid := strings.Index(line, "](")
	if mid < 2 || !strings.HasSuffix(line, ")") {
		return "", "", false
	}
	alt := line[2:mid]
	src := strings.TrimSpace(line[mid+2 : len(line)-1])
	if src == "" {
		return "", "", false
	}
	return alt, src, true
}

func renderInlineMarkdown(text string) string {
	var out strings.Builder
	for len(text) > 0 {
		start := strings.Index(text, "`")
		if start < 0 {
			out.WriteString(html.EscapeString(text))
			break
		}
		out.WriteString(html.EscapeString(text[:start]))
		text = text[start+1:]
		end := strings.Index(text, "`")
		if end < 0 {
			out.WriteString(html.EscapeString("`" + text))
			break
		}
		out.WriteString("<code>")
		out.WriteString(html.EscapeString(text[:end]))
		out.WriteString("</code>")
		text = text[end+1:]
	}
	return out.String()
}

func uniqueHeadingID(text string, slugCounts map[string]int) string {
	base := slugifyHeadingText(text)
	if base == "" {
		base = "section"
	}
	count := slugCounts[base]
	slugCounts[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, count+1)
}

func slugifyHeadingText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	var out strings.Builder
	lastDash := false
	for _, r := range text {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			out.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '/' || r == '.':
			if !lastDash && out.Len() > 0 {
				out.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(out.String(), "-")
}

func (s *Server) handleDocsCatalog(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	writeJSON(w, map[string]any{"docs": availableUIDocs()})
}

func (s *Server) handleDocsView(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}

	doc, err := resolveUIDoc(r.URL.Query().Get("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	body, err := os.ReadFile(doc.path)
	if err != nil {
		http.Error(w, "read doc failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	content := string(body)
	contentHTML, headings := renderMarkdownDocument(content)
	writeJSON(w, map[string]any{
		"key":          doc.spec.Key,
		"name":         doc.spec.Filename,
		"label":        doc.spec.Label,
		"description":  doc.spec.Description,
		"path":         doc.path,
		"content":      content,
		"content_html": contentHTML,
		"headings":     headings,
		"rendered":     "html",
	})
}

func (s *Server) handleDocsBrowser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cfg := s.currentConfig()
	page := strings.ReplaceAll(docsBrowserHTML, "__KNIT_TOKEN__", cfg.ControlToken)
	_, _ = w.Write([]byte(page))
}

func (s *Server) handleDocsAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathSuffix := strings.TrimPrefix(r.URL.Path, "/docs/assets/")
	assetPath, err := resolveUIAssetFile(pathSuffix)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, assetPath)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	assetPath, err := resolveUIAssetFile("favicon.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, assetPath)
}
