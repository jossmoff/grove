# Turning these docs into a website (optional)

These docs are plain Markdown so they render on GitHub today with no build step.
If grove ever outgrows that — versioned docs, search, a landing page — the
content here is already structured to drop into a static-site generator with no
rewrite.

The recommended path, matching the herdr project's approach, is
[Astro Starlight](https://starlight.astro.build/):

```sh
npm create astro@latest -- --template starlight docs-site
# move docs/*.md into docs-site/src/content/docs/ (rename .md → .md or .mdx)
# each file's H1 becomes the page title; add frontmatter title/description
```

Then serve it with GitHub Pages via an Actions workflow. **Do not build this
until the tool has earned it** — a docs site with its own `package.json`, build,
and deploy pipeline is exactly the speculative over-investment grove's own
design philosophy warns against (see [decisions](decisions.md)). Markdown on
GitHub is the right amount of docs infrastructure for a tool at grove's maturity.

The frontmatter Starlight expects, for when the time comes:

```markdown
---
title: Concepts
description: The grove vocabulary — grove, role, manifest, profile.
---
```
