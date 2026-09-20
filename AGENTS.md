# Blog authoring guide

These instructions apply to the entire repository. Use them whenever creating or editing content in `content/posts/`.

## Source of truth

- `README.md`: setup, local preview, Docker, and publishing workflow.
- `archetypes/default.md`: canonical post front matter.
- `hugo.yaml`: permalinks, taxonomies, table of contents, Goldmark, Chroma, and PaperMod behavior.
- `assets/css/extended/code.css`: project-specific code-block styling.
- `layouts/_default/_markup/render-codeblock.html`: fenced-code render hook and `filename` support.
- `Dockerfile`: production build and runtime behavior.

Do not duplicate configuration in a post. If behavior is unclear, inspect these files before inventing syntax or conventions.

## Creating a post

- Store each post as a Hugo leaf bundle in a kebab-case directory: `content/posts/my-post/index.md`.
- Keep post-specific images and other media in `content/posts/my-post/assets/`, and supporting source material in `content/posts/my-post/code/`. Do not put secrets or private material in either directory because Hugo publishes page resources.
- Use the following YAML front matter and keep new posts as drafts unless publication is explicitly requested:

```yaml {linenos=false}
---
title: "Clear, specific article title"
date: 2026-09-16T00:00:00-03:00
draft: true
description: "One concise sentence describing what the reader will learn."
tags: ["go", "grpc"]
categories: ["engineering"]
---
```

- Use an RFC 3339 timestamp with an explicit timezone.
- Keep the description useful in lists and search results; do not repeat the title verbatim.
- Use short, lowercase taxonomy values and reuse existing tags/categories when applicable.
- Do not add a body-level `#` heading: Hugo renders the front-matter title as the page title.

## Article structure

- Open with a brief paragraph that states the problem and expected takeaway.
- Start body sections at `##`; use `###` for subsections. Hugo's table of contents includes levels 2–3.
- Prefer descriptive headings over generic labels such as “Overview” or “Details.”
- Explain why an example matters before or immediately after showing it.
- End with a concise conclusion, practical takeaway, or next step rather than repeating every section.
- Keep paragraphs focused and use lists or tables only when they improve scanning.

## Supported Markdown

Prefer portable Markdown that Goldmark and PaperMod render without custom components:

- paragraphs and emphasis
- ordered and unordered lists
- blockquotes
- tables
- descriptive links
- images with meaningful alt text
- fenced code blocks

Do not add raw HTML: `hugo.yaml` sets Goldmark `unsafe: false`. Do not assume Mermaid, custom shortcodes, or client-side JavaScript renderers exist unless the repository is explicitly updated to support them.

## Code blocks and CSS integration

Hugo Extended renders fenced code through Chroma at build time. Always provide a language identifier; syntax guessing is disabled.

````md
```go {filename="cmd/server/main.go",linenos=inline,hl_lines=[3,"5-6"],linenostart=20}
func main() {
    run()
}
```
````

The `filename` attribute is handled by the custom render hook. It wraps highlighted output in `.code-block` and renders a `.code-block__filename` header styled by `assets/css/extended/code.css`.

- Use `filename` when the file path gives useful context.
- Use `hl_lines` sparingly and discuss the highlighted lines in the prose.
- Use `linenostart` for an excerpt whose original line numbers matter.
- Line numbers are enabled globally. Add `linenos=false` for short shell commands, payloads, or tiny snippets where numbers distract.
- Use `linenos=inline` when selecting explicit Chroma line-number behavior; the CSS keeps numbers and code in one horizontal scroll area.
- Keep lines reasonably short even though code blocks scroll horizontally.
- Common identifiers include `go`, `python`, `java`, `dockerfile`, `yaml`, `json`, `bash`, `sql`, `typescript`, `tsx`, `html`, `css`, and `text`.
- Label incomplete code as illustrative or pseudocode. Do not present non-runnable fragments as complete programs.

For commands, prefer a compact fence without line numbers:

````md
```bash {linenos=false}
go test ./...
```
````

## Technical writing standards

- Write concise, direct technical prose in English unless another language is requested.
- Preserve factual boundaries: distinguish verified behavior, design proposals, and simplified examples.
- Never invent benchmark results, security guarantees, production details, or API contracts.
- Define uncommon acronyms on first use.
- Use consistent terminology and capitalization throughout the article.
- Use descriptive link text instead of “click here.”
- Do not expose secrets, private endpoints, customer data, credentials, or internal identifiers.

## Validation checklist

Before finishing a content change:

1. Confirm the filename and all required front-matter fields.
2. Confirm a new post remains `draft: true` unless publication was requested.
3. Check heading order, links, image alt text, fence languages, and balanced fences.
4. Run `git diff --check`.
5. Run `make build` to validate Hugo rendering.
6. If the pinned toolchain is missing, run `make setup` once and retry.
7. Inspect `git status --short` and ensure only intended files changed.

Use `make serve` for a draft-inclusive local preview at `http://localhost:1313`. Docker users can run `docker compose up --build`. The production image builds the site with Hugo Extended and serves only generated files from BusyBox; source Markdown is not present in the final image.
