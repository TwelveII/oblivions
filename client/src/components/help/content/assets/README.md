# Help center images

Drop screenshots and demos here. They are bundled with the client (Vite-hashed
URLs), so they work identically in Electron, the web app, and the landing page —
offline, no server, no CORS.

## Naming

One language-agnostic set, referenced by `<slug>-<N>.<ext>` where `<slug>` is the
article filename (without `.md`) and `<N>` is the placeholder order in that article.

Examples: `commands-1.webp`, `screen-share-2.webp`, `split-view-1.gif`.

Both `en/<slug>.md` and `tr/<slug>.md` reference the **same** file at the same
position, so one image serves both languages.

## Formats

| Content | Use | Avoid |
| --- | --- | --- |
| Static screenshot | `.webp` (~50–150 KB) | `.png` (3–5× larger) |
| Motion demo | `.webm` / `.mp4` or animated `.webp` | `.gif` (10× larger) |

`.webp` / `.png` / `.gif` render as `<img>`; `.webm` / `.mp4` render as a muted,
looping `<video>`. Pick the extension in the markdown reference to match the file.

## Wiring up an image

Articles currently carry a placeholder line, e.g.:

```
> 📸 **Screenshot:** `assets/commands-1.webp` — description of the shot.
```

When the image exists, replace that line with a real reference:

```
![description of the shot](assets/commands-1.webp)
```

The renderer resolves `assets/<file>` to the bundled URL automatically.
