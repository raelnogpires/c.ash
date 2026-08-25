# Design — [c]ash

A locked visual system for the desktop application. Every view uses the same
geometry, typography, interaction language, and information hierarchy. The
light, dark, and gothic themes change colour tokens only.

## Genre

Modern-minimal utility: a calm personal ledger with the precision of a
financial tool and the warmth required by someone organising money for the
first time.

## Macrostructure family

- App views: **Stat-Led**. The number that answers the view’s main question is
  the visual anchor; supporting data follows as hairline-separated records.
- Onboarding: **Split Studio**. Product promise and setup form share a diptych.
- Dialogs: compact work surfaces with one clear action and no decorative layer.

App components may use H4 Stat-Led, F3 Tabular Spec Sheet, C1 Outlined Chip,
and N3 Side Rail. App views do not use hero imagery, decorative footers, or
marketing enrichment.

## Themes

The existing colours are preserved perceptually and expressed as OKLCH tokens:

- **Light** — warm parchment, soft ivory surfaces, forest ink and accent.
- **Dark** — green-black paper, lifted moss surfaces, mint accent.
- **Gothic** — charcoal paper, burgundy rules, dusty-rose accent.

Theme changes never move, resize, or rename a control. Positive values remain
green; expenses and alerts remain red; icons, labels, and signs repeat every
colour-coded meaning.

## Typography

- Display and body: bundled Inter Variable, roman, weights 400–600.
- Financial values: Inter Variable, weights 500–600, tabular figures.
- Display tracking: `-0.035em`; body tracking remains neutral.
- Type scale: major third; the lead balance uses `--text-stat` and tabular figures.
- Headings are always roman. Body copy stays between 45 and 75 characters where
  it forms a paragraph.

Inter is packaged with the application, so the local-first interface remains
network-free and visually consistent across platforms.

## Iconography and hierarchy

- Lucide is the only interface icon family and is compiled into the application.
- Icons use a consistent 1.8 stroke and never replace an accessible control name.
- Income, expense, transfer, visibility, navigation, privacy, and empty states
  use semantic Lucide glyphs rather than typographic characters.
- Every view has one page title. Internal headings remain only when they add new
  context; labels and footers never repeat the active destination.

## Spacing

A named 4-point scale lives in `frontend/tokens.css`. Layout uses varied gaps
from that scale; card padding, page padding, and section rhythm are deliberately
not identical.

## Motion

- Easings: `--ease-out`, `--ease-in`, and `--ease-in-out` only.
- Motion primitives: button press, menu/dialog entrance, and undo snackbar.
- Only `transform` and `opacity` animate.
- Reduced motion removes spatial movement and collapses feedback to 1 ms.

## Microinteractions stance

- Success is silent when the result is already visible.
- Reversible transaction removal is optimistic and offers Undo for eight seconds.
- Destructive account removal retains explicit confirmation.
- Focus is immediate and visible; hover is never the only affordance.

## CTA voice

- Primary: compact forest/mint/rose fill when the action must dominate.
- Secondary: outlined rectangular control with a direct verb.
- Labels stay on one line and controls share a 44 px minimum height.

## Per-view allowances

- Dashboard may use the large stat, line chart, and proportional account bars.
- Data-management views use open lists and hairline rules before cards.
- Onboarding may use one typographic background mark; no photography or stock art.
- Settings may preview all three palettes while preserving identical geometry.

## What views MUST share

- `[c]ash` wordmark and its bracket motif.
- The light, dark, and gothic palette families.
- Inter typography, Lucide iconography, 4-point spacing scale, control height,
  focus ring, and button language.
- A compact collapsible rail on desktop and an icon dock at narrow widths.
- Accessible loading, empty, error, disabled, success, and undo feedback.

## What views MAY differ on

- The dominant stat or question answered by the view.
- Whether supporting content is a ledger, a chart, a form, or a palette preview.
- Section spacing and column proportions when the content requires it.

## Exports

`frontend/tokens.css` is the runtime source of truth and contains every theme.
The following mappings make the core system portable.

### tokens.css

```css
@import "./frontend/tokens.css";
```

### Tailwind v4 `@theme`

```css
@theme {
  --color-paper: oklch(95.83% 0.0111 89.72);
  --color-paper-2: oklch(99.42% 0.0069 88.64);
  --color-paper-3: oklch(93.14% 0.0140 88.69);
  --color-ink: oklch(26.84% 0.0189 160.95);
  --color-muted: oklch(51.50% 0.0152 159.58);
  --color-rule: oklch(88.57% 0.0142 88.69);
  --color-accent: oklch(37.86% 0.0597 165.03);
  --color-focus: oklch(53.24% 0.0794 163.94);
  --font-display: "Inter Variable", Inter, ui-sans-serif, system-ui, sans-serif;
  --font-body: "Inter Variable", Inter, ui-sans-serif, system-ui, sans-serif;
  --spacing-3xs: 0.25rem;
  --spacing-2xs: 0.5rem;
  --spacing-xs: 0.75rem;
  --spacing-sm: 1rem;
  --spacing-md: 1.25rem;
  --spacing-lg: 1.5rem;
  --spacing-xl: 1.75rem;
  --spacing-2xl: 2rem;
  --spacing-3xl: 2.5rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-md: 1.25rem;
  --text-lg: 1.5625rem;
  --radius-card: 0.75rem;
  --radius-input: 0.5rem;
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
}
```

### DTCG `tokens.json`

```json
{
  "$schema": "https://design-tokens.github.io/community-group/format/",
  "color": {
    "paper": { "$value": "oklch(95.83% 0.0111 89.72)", "$type": "color" },
    "paper-2": { "$value": "oklch(99.42% 0.0069 88.64)", "$type": "color" },
    "paper-3": { "$value": "oklch(93.14% 0.0140 88.69)", "$type": "color" },
    "ink": { "$value": "oklch(26.84% 0.0189 160.95)", "$type": "color" },
    "muted": { "$value": "oklch(51.50% 0.0152 159.58)", "$type": "color" },
    "rule": { "$value": "oklch(88.57% 0.0142 88.69)", "$type": "color" },
    "accent": { "$value": "oklch(37.86% 0.0597 165.03)", "$type": "color" },
    "focus": { "$value": "oklch(53.24% 0.0794 163.94)", "$type": "color" }
  },
  "font": {
    "display": { "$value": "Inter Variable, Inter, ui-sans-serif, system-ui, sans-serif", "$type": "fontFamily" },
    "body": { "$value": "Inter Variable, Inter, ui-sans-serif, system-ui, sans-serif", "$type": "fontFamily" }
  },
  "space": {
    "xs": { "$value": "0.75rem", "$type": "dimension" },
    "sm": { "$value": "1rem", "$type": "dimension" },
    "md": { "$value": "1.25rem", "$type": "dimension" },
    "lg": { "$value": "1.5rem", "$type": "dimension" },
    "xl": { "$value": "1.75rem", "$type": "dimension" },
    "2xl": { "$value": "2rem", "$type": "dimension" },
    "3xl": { "$value": "2.5rem", "$type": "dimension" }
  },
  "duration": {
    "micro": { "$value": "120ms", "$type": "duration" },
    "short": { "$value": "220ms", "$type": "duration" },
    "long": { "$value": "420ms", "$type": "duration" }
  }
}
```

### shadcn/ui CSS variables

```css
:root {
  --background: 95.83% 0.0111 89.72;
  --foreground: 26.84% 0.0189 160.95;
  --card: 99.42% 0.0069 88.64;
  --card-foreground: 26.84% 0.0189 160.95;
  --popover: 98.68% 0.0118 79.79;
  --popover-foreground: 26.84% 0.0189 160.95;
  --primary: 37.86% 0.0597 165.03;
  --primary-foreground: 95.83% 0.0111 89.72;
  --secondary: 93.14% 0.0140 88.69;
  --secondary-foreground: 26.84% 0.0189 160.95;
  --muted: 88.57% 0.0142 88.69;
  --muted-foreground: 51.50% 0.0152 159.58;
  --destructive: 51.16% 0.1301 26.77;
  --destructive-foreground: 98.68% 0.0118 79.79;
  --border: 88.57% 0.0142 88.69;
  --input: 81.76% 0.0145 88.70;
  --ring: 53.24% 0.0794 163.94;
  --radius: 0.75rem;
}
```
