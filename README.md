# Developer Blog

A lightweight, self-hosted developer blog built with [Hugo Extended](https://gohugo.io/) and [PaperMod](https://github.com/adityatelange/hugo-PaperMod). Posts are Markdown files committed to Git; there is no database or application server.

The project-local toolchain uses **Go 1.27.1**. Docker downloads only the pinned Hugo Extended binary to compile the site, then serves the generated files with BusyBox's tiny built-in HTTP server.

## Quick start (Docker)

Start the live-reloading development server:

```sh
docker compose up --build
```

Open <http://localhost:1313>. Changes to `content/`, `assets/`, `layouts/`, and `hugo.yaml` reload automatically.

Create an optimized production image:

```sh
docker build \
  --build-arg HUGO_BASEURL=https://blog.example.com/ \
  -t developer-blog:latest .
docker run --rm -p 8080:8080 developer-blog:latest
```

The published site is available at <http://localhost:8080>. Set `HUGO_BASEURL` to the final public URL before publishing, including a trailing slash.

## Local development without Docker

Install Git, `curl`, and `tar`, then install the pinned project-local toolchain:

```sh
make setup
```

This downloads Go 1.27.1 and the official Hugo **Extended** binary into `.tools/`; it does not alter your system-wide Go installation or require a compiler. Hugo's extended edition is required for the theme's CSS processing. Then:

```sh
make serve
```

`make theme` fetches the pinned PaperMod release into the ignored `themes/` directory. `make build` writes the static site to `public/`. `make serve` and `make build` tell you to run `make setup` if the local toolchain has not been prepared yet.

## Write a post

```sh
hugo new content/posts/my-post.md
```

Update the front matter, change `draft: true` to `false`, and write Markdown. The starter post at `content/posts/hello-developer-blog.md` demonstrates Go, Python, Java, Dockerfile, YAML, and shell fences.

Code is highlighted at build time by Hugo's Chroma integration: no browser-side syntax-highlighting library is shipped. Line numbers are enabled globally, and individual fences may refine them. Add `filename` to show a file label above the snippet:

````md
```go {filename="cmd/server/main.go",linenos=inline,hl_lines=[3,"5-6"],linenostart=20}
func main() {
    // highlighted output
}
```
````

Useful Chroma language names include `go`, `python`, `java`, `dockerfile`, `yaml`, `json`, `bash`, `sql`, `typescript`, `tsx`, `html`, and `css`.

## Project layout

```text
content/             Markdown pages and posts
archetypes/          Front-matter template for new posts
assets/css/extended/ Small PaperMod-compatible Chroma refinements
layouts/             Small compatibility override for PaperMod v8 on current Hugo
hugo.yaml            Site, PaperMod, and Chroma configuration
Dockerfile           Hugo Extended build plus a tiny BusyBox static server
compose.yaml         Live-reload development service
```

## Publish

Build the image above and expose port 8080 directly, or place it behind an existing reverse proxy if you already use one. The final image contains only BusyBox and `public/`, never source Markdown, Go tooling, a database, or runtime dependencies.

Before publishing, update `baseURL`, `title`, `params.homeInfoParams`, social links, and the `menu` in `hugo.yaml`.
