---
name: website-docs
description: Use when adding or editing a page of the nanci website (Hugo) or refreshing screenshots. Covers the Hextra setup, front matter, the local layout overrides and shortcodes, link rules for the HTML and Markdown copies, screenshot generation, the Pages workflow, a local build with a link check, and what belongs on the site versus docs/.
---

# Website docs: the Hugo site under `website/`

The site at https://vasfvitor.github.io/nanci/ is the user documentation. Technical detail lives in `docs/*.md` on GitHub and the site links to it on purpose; never copy one into the other (`docs/DEVELOPMENT.md`, "Site de documentação"). Open each cited file before copying its pattern.

## Setup

- `website/hugo.toml`: theme Hextra as a Hugo module (`github.com/imfing/hextra`, pinned `v0.12.3` in `website/go.mod`), `baseURL` `https://vasfvitor.github.io/nanci/` (the site lives under `/nanci/`), `defaultContentLanguage = "pt"`, `enableGitInfo = true` (last-modified dates come from git), `markup.goldmark.renderer.unsafe = true` (raw HTML in content, used by the home page).
- Outputs: `home = ["html", "rss", "llms"]`, `section` and `page` add `"markdown"`. Hextra defines both formats (`hugo config --source website` shows `outputformats.markdown` with `ugly = true`), so every docs page also ships as `docs/<page>.md` and the root gets `llms.txt` (`layouts/llms.txt`, one line per page from its `description`; a page opts out with `llms: false`). `_partials/custom/head-end.html` links the Markdown copy with `rel="alternate"` and, on the home page, `llms.txt`.
- Upgrading Hextra (`hugo mod get -u github.com/imfing/hextra` in `website/`): re-diff the three copied templates against the new theme: `layouts/docs/single.html`, `layouts/page.markdown.md`, `layouts/llms.txt`.

## Content

- Pages are `website/content/docs/*.md`, in Portuguese. Voice: short sentences, second person ("Você precisa de..."), the action first, UI labels in bold exactly as the app shows them (`**Registrar ciência**`), no marketing, no rule explanations the app already enforces (`65881e0` cut the site to what users act on).
- Front matter: `title`, `description` (one sentence; feeds `llms.txt` and the meta description), `weight`. Optional: `fonte: { nome, url }` (official portal) and `referencia: "docs/NFE_SEFAZ.md"` (developer doc), both shown by `_partials/page-fields.html` in the field strip under the title; `toc: false` (as `faq.md`); `aliases` when a page is renamed (`nfse.md` keeps `/docs/nfse-adn/`).
- Weight order today: `docs/_index.md` 10, `instalacao` 20, `nfse` 30, `nfe` 31, `cte` 32, `cli` 40, `certificados` 50, `privacidade` 60, `faq` 70, `troubleshooting` 80, `desenvolvimento` 90. A new source page goes in the 3x band, a new topic between the existing tens.
- `content/_index.md` is the home (`layout: "hextra-home"`): raw HTML with `nc-` classes, `{{< danfe-sheet >}}` behind the hero, one `nc-box` per document type. A new document type needs a box there and a row in the table of `docs/_index.md`.
- `docs/_index.md` holds the source table, Hextra `{{< cards >}}`/`{{< card link="..." >}}`, and "Fora do escopo".

## Layouts and shortcodes

- `layouts/docs/single.html`: Hextra's docs template plus `{{ partial "page-fields.html" . }}` under the title. `page-fields.html` shows "Atualizado em" (git `Lastmod`, linked to the file's commit history), "Fonte oficial" and "Referência técnica" (a `blob/main/<referencia>` link).
- `layouts/page.markdown.md`: Hextra's Markdown copy, with `](../` rewritten to `](/nanci/docs/`. The copy is served at `docs/<page>.md`, one level above the HTML at `docs/<page>/`, so `../nfe/` would resolve to `/nanci/nfe/` and 404 (issue #14, fixed in `476cb2f`). Shortcodes stay as written in the copy.
- `callout`: `type` is `warning` (caption "Atenção"), `error` ("Cuidado"), `important` ("Importante"), `info` or omitted ("Nota"); the body is Markdown.
  ```
  {{< callout type="warning" >}}
  Uma manifestação enviada não pode ser cancelada.
  {{< /callout >}}
  ```
- `theme-image`: `light` (required), `dark` (optional; without it the light image shows in both themes), `alt` (also the figure caption on docs pages; the home renders no figure), `class`, `width`, `height`. Paths starting with `/` go through `relURL`, so write `/img/screenshots/...`.
  ```
  {{< theme-image light="/img/screenshots/cte-light.png" dark="/img/screenshots/cte-dark.png" alt="Página de CT-e" >}}
  ```
- `faq`: `{{< faq "Pergunta?" >}}Resposta.{{< /faq >}}`; the block id is `anchorize` of the question, so other pages link `../faq/#emito-nfs-e-mas-o-nanci-não-baixou-nada`.
- `danfe-sheet`: home background only.
- `assets/css/custom.css` (Hextra loads it): every class is prefixed `nc-`; colors are tokens (`--nc-ink`, `--nc-text`, `--nc-caption`, `--nc-rule`, `--nc-paper`, `--nc-field`, `--nc-blue`, `--nc-blue-text`) defined in `:root` and redefined in `.dark`, next to Hextra's `--primary-*` and `--hx-color-*`. New rules use the tokens, not hex values. Templates use Hextra's `hx:`-prefixed utility classes.

## Links

- Between docs pages: relative, `../nfe/` or `../nfe/#limite-de-consultas`. Works in the HTML and is rewritten for the Markdown copy. From `docs/_index.md`: `nfse/`. From the home: `docs/nfse/`.
- Never root-absolute (`/docs/nfe/` drops the `/nanci/` prefix) and never a `github.com/.../website/content` link between site pages. Only Markdown `](../` is rewritten; avoid `href="../"` in raw HTML on docs pages.
- Anchors are Hugo's heading ids: lowercase, accents kept, spaces to `-` (`#manifestação-do-destinatário`).
- Developer docs are linked on purpose and only as GitHub URLs: the list in `desenvolvimento.md` and the `referencia` field. In the other direction, `docs/*.md` link the site by its full URL (`docs/CERTIFICATES.md`).
- External facts cite the official source inline (NT links on `nfe.fazenda.gov.br`/`cte.fazenda.gov.br`, Ajustes on `confaz.fazenda.gov.br`, `gov.br/nfse`); see the `research-doc` skill.

## Screenshots

- Generated from mocks by `internal/desktop/frontend/scripts/generate-screenshots.ts`: Vite on port 5555 with a fake `window.go`, Playwright Chromium at 1280x800 and `deviceScaleFactor: 2`, one spec per `name` and `theme`, written to `docs/screenshots/<name>-<theme>.png`.
- `pnpm run screenshots` (or `make screenshots`) runs all; `pnpm run screenshots nfe cte` limits the run to those router paths (`/`, `documents`, `credentials`, `query`, `settings`, `nfe`, `cte`; leading slash optional). Specs with `fitsWidth: true` fail when the document table scrolls sideways.
- A new screenshot is a spec in the `screenshots` array, normally a light and a dark entry. `dialogo-ciencia-nfe`, `dialogo-eventos-cte` and `nfe-bloqueada` are light only, so their `theme-image` gets no `dark`. Wails methods the page calls need a mock there (see `desktop-method`).
- `website/scripts/copy-screenshots.sh` copies `docs/screenshots/*.png` to `website/static/img/screenshots/` (gitignored). The README uses `docs/screenshots/` directly.

## Publish and build

- `.github/workflows/website.yml` runs on push to `main` touching `website/**` or `docs/screenshots/**`, and by hand: Go 1.26 for the module, Hugo 0.163.2 extended (`peaceiris/actions-hugo`), the copy script, `hugo --minify`, then GitHub Pages.
- Local build (Hugo extended and Go on the PATH; `public/` is gitignored, but build outside the tree anyway):
  ```bash
  bash website/scripts/copy-screenshots.sh
  hugo --source website --destination "$TMP/site"   # or: hugo server --source website  -> http://localhost:1313/nanci/
  ```
- Link check on the build (what #14 needed; both must print nothing):
  ```bash
  S="$TMP/site"
  grep -l '](\.\./' "$S"/docs/*.md
  grep -oh '](/nanci/docs/[^)]*)' "$S"/docs/*.md | sort -u | sed 's#](/nanci/##; s#)$##' |
    while IFS='#' read -r p a; do
      [ -f "$S/${p}index.html" ] || echo "missing page $p"
      [ -z "$a" ] || grep -q "id=\"$a\"" "$S/${p}index.html" || echo "missing anchor $p#$a"
    done
  ```

## Checklist

- [ ] Front matter: `title`, one-sentence `description`, `weight` in order; `fonte`/`referencia` on source pages; `aliases` if renamed
- [ ] Portuguese, short, second person; UI labels in bold as the app shows them (grep the `.vue` file)
- [ ] Shortcodes `callout`, `theme-image`, `faq` with the parameters above
- [ ] Screenshots regenerated (`pnpm run screenshots <route>`) when the UI shown changed; light and dark pair unless the spec is light only
- [ ] `cli.md` command table updated when a command was added, renamed or removed (`Use:` in `internal/cli/*.go`)
- [ ] Home box and `docs/_index.md` table updated for a new document type
- [ ] Links relative (`../page/#anchor`); no root-absolute or GitHub links between site pages; developer docs only via `referencia` or `desenvolvimento.md`
- [ ] Local build and link check clean; `llms.txt` lists the page with its description
- [ ] Commit as `website: <change>` (`docs:` when `docs/` changes too); LF
