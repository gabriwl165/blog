# Blog authoring guide

Use these rules when creating or editing files in `content/posts/`.

## Post format

- Store each post as a kebab-case Markdown file: `content/posts/my-post.md`.
- Use YAML front matter with `title`, `date`, `draft`, `description`, `tags`, and `categories`.
- Keep new posts as `draft: true` unless publication is explicitly requested.
- Use an RFC 3339 date with timezone, for example `2026-09-16T00:00:00-03:00`.
- Start body sections at `##`; use `###` for subsections so Hugo's table of contents includes them.
- Prefer standard Markdown: paragraphs, lists, tables, blockquotes, links, and fenced code blocks.
- Do not add raw HTML. Goldmark renders with `unsafe: false`.

## Code blocks

Hugo Extended renders fenced code through Chroma at build time. Always specify the language:

````md
```go {filename="cmd/server/main.go",linenos=inline,hl_lines=[3,"5-6"],linenostart=20}
func main() {}
```
````

- Use `filename` when the source filename adds context; the custom render hook and CSS add its header.
- Use `hl_lines` only to direct attention to lines discussed in the prose.
- Use `linenostart` when showing an excerpt from a larger file.
- Line numbers are globally enabled; use `linenos=false` for short commands or payloads where numbers distract.
- Common language identifiers include `go`, `python`, `java`, `dockerfile`, `yaml`, `json`, `bash`, `sql`, `typescript`, `tsx`, `html`, and `css`.

## Writing and validation

- Write concise technical prose and explain why a snippet matters before or after it.
- Keep examples executable or label them clearly as illustrative or pseudocode.
- Use descriptive link text and meaningful image alt text.
- Preview drafts with `make serve` and validate the final rendering with `make build`.
- Run `make setup` first when the pinned Hugo Extended toolchain is unavailable.
- Before finishing, run `git diff --check` and verify that only intended files changed.

Refer to `README.md` for setup and publishing commands, `hugo.yaml` for rendering behavior, and `assets/css/extended/code.css` plus `layouts/_default/_markup/render-codeblock.html` for code-block presentation.
