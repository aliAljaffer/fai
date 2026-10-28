# fai brand

## Files

| File | Use |
|---|---|
| `fai-icon.svg` | App icon, favicon, avatar. Amber tile with the f cut in ink. Used in the app nav at 20 px. |
| `fai-symbol.svg` | The f on its own, ink, for print, stamps and single-colour use. |
| `fai-symbol-reversed.svg` | Same, white, for dark or photographic backgrounds. |
| `fai-wordmark.svg` | "fai" only, where the icon would repeat the f (documents, footers). |
| `fai-lockup.svg` | Icon plus wordmark, light backgrounds. |
| `fai-lockup-reversed.svg` | Icon plus wordmark, dark backgrounds. |
| `readme-header.svg` | Centred lockup banner for the repo README. Render: `renders/readme-header.png`. |

Renders for review live in `renders/`.

## Colour

| Token | Value | Use |
|---|---|---|
| Slate | `#ACC7E4` | Icon tile only. Never as a large background. |
| Ink | `#17130E` | The f, wordmark, text on slate. |
| Paper | `#FAFCFE` | Light background. |
| Night | `#0D1012` | Dark background. |

In the app these map to CSS variables in `frontend/src/index.css`: `--primary` is a deep slate in light mode (`oklch(0.42 0.05 250)`, 8.4:1 on paper) and a light slate in dark mode (`oklch(0.82 0.05 250)`, 11:1 on the dark background). Backgrounds and text carry a trace of the same hue so the UI belongs to the mark.

Labels and statuses live in `frontend/src/lib/palette.ts`: eight label hues at a single lightness and chroma, plus three status colours (todo slate, doing teal, done green). Status is shown as a dot so the text keeps full contrast. `@task` references use the brand slate.

Semantic tokens sit beside the base palette in `index.css` and replace literal colours: `--attention` (reminders, sync warnings), `--attention-foreground`, `--destructive-foreground`, `--code-string`, `--code-number`, the `--chart-1..2` slate ramp, and `--brand-tile` / `--brand-ink` for the mark itself.

## Typography

The UI is set in Figtree (Google Fonts), self-hosted through `@fontsource-variable/figtree`. The wordmark is not Figtree: it is custom stroke geometry built to match the f symbol, so it stays fixed whatever the UI typeface is.

## Rules

- Clear space on all sides: one crossbar width (the f's horizontal bar).
- Minimum sizes: icon 16 px, symbol 24 px wide, lockup 96 px wide.
- The f is always either ink on light or white on dark. Slate is the tile colour, not a substitute for ink.
- Do not rotate, outline, stretch, add gradients, or place the ink f on a mid-tone background.

## Notes

- All letterforms are drawn as paths. No fonts are embedded.
- The marks are stroke-based. If you need expanded outlines for cutting or a vinyl plotter, convert the strokes in a vector editor first.
- No trademark search has been done.
