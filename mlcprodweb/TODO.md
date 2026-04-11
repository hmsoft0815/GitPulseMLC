# MLCgitChecker — Product Page Authoring Guide

This directory contains everything needed to generate the product marketing page.
Run `mlcprodweb preview` from the project root to see it locally.

---

## File overview

| File / Dir     | Purpose |
|----------------|---------|
| `meta.yaml`    | Structured metadata: version, platforms, features, download links |
| `en.md`         | English marketing copy (H1 = hero title, rest = body) |
| `de.md`         | German marketing copy |
| `assets/`       | Screenshots and product images referenced in `meta.yaml` |
| `pages/`        | Optional sub-pages (docs, examples, changelog, …) |

---

## meta.yaml fields

```yaml
id:        "my-product"         # slug used in URL: /products/my-product/
name:      "My Product"
version:   "1.2.0"
status:    "stable"             # beta | stable
category:  "Development Tools"
license:   "MIT"
platforms: [Windows, macOS, Linux]
github:    "https://github.com/..."

screenshots:
  - "hero.png"                  # files must exist in assets/
  - "screenshot2.png"

downloads:
  free: false                   # false = login required (auth gate in JS)
  windows: "myproduct-win.exe"  # filename at /downloads/my-product/<file>
  linux:   "myproduct-linux"
  macos:   "myproduct-mac.dmg"

features:
  - icon:     "zap"             # Lucide icon name (https://lucide.dev/icons)
    title_en: "Fast"
    title_de: "Schnell"
    desc_en:  "One sentence."
    desc_de:  "Ein Satz."
```

---

## Writing en.md / de.md

- The **first line** must be a Markdown H1 (`# Title`) — this becomes the hero heading.
- Everything after is rendered as the page body.
- Use standard Markdown: headings, lists, bold, code blocks.
- Keep the copy focused on **benefits**, not feature lists (features go in meta.yaml).

---

## Adding screenshots

1. Put image files in `assets/` (PNG, WebP, SVG).
2. Reference them in `meta.yaml` under `screenshots`.
3. Run `mlcprodweb preview` to see them.

---

## Sub-pages (docs, examples, …)

Create `pages/<slug>/en.md` and `pages/<slug>/de.md`:

```
pages/
  docs/
    en.md   ← /products/my-product/en/docs/
    de.md   ← /products/my-product/de/docs/
  examples/
    en.md
    de.md
```

Sub-pages appear automatically in the product nav. H1 becomes the page title.

---

## Downloads

- `downloads.free: true` → direct `<a href>` links (rendered at build time).
- `downloads.free: false` → auth-gated: `auth.js` shows download buttons only to logged-in users.
  Upload files to VPS: `rsync ./myproduct-win.exe root@mlcgo.eu:/var/www/html/downloads/my-product/`

---

## Publishing to mlcgo.eu

1. Commit this `mlcprodweb/` directory.
2. In the `mlcgo-vps` repo, add this project to `sync.yaml`.
3. Run `task content:sync && task products:deploy`.
