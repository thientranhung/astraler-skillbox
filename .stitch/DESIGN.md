---
name: Astraler Skillbox
colors:
  surface: '#F9FAFB'
  surface-dim: '#D4D4D8'
  surface-bright: '#FFFFFF'
  surface-container-lowest: '#FFFFFF'
  surface-container-low: '#F4F4F5'
  surface-container: '#ECECEE'
  surface-container-high: '#E4E4E7'
  surface-container-highest: '#DDDDE0'
  on-surface: '#09090B'
  on-surface-variant: '#71717A'
  inverse-surface: '#27272A'
  inverse-on-surface: '#F9FAFB'
  outline: '#71717A'
  outline-variant: '#E4E4E7'
  surface-tint: '#E11D48'
  primary: '#E11D48'
  on-primary: '#FFFFFF'
  primary-container: '#FFF1F2'
  on-primary-container: '#9F1239'
  inverse-primary: '#FDA4AF'
  secondary: '#6B7280'
  on-secondary: '#FFFFFF'
  secondary-container: '#F4F4F5'
  on-secondary-container: '#374151'
  tertiary: '#F59E0B'
  on-tertiary: '#FFFFFF'
  tertiary-container: '#FFFBEB'
  on-tertiary-container: '#92400E'
  error: '#EF4444'
  on-error: '#FFFFFF'
  error-container: '#FEF2F2'
  on-error-container: '#DC2626'
  success: '#22C55E'
  on-success: '#FFFFFF'
  success-container: '#F0FDF4'
  on-success-container: '#15803D'
  background: '#F9FAFB'
  on-background: '#09090B'
  surface-variant: '#E4E4E7'
typography:
  page-title:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Inter", sans-serif'
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.01em
  section-header:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Inter", sans-serif'
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: '0'
  body:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Inter", sans-serif'
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: '0'
  caption:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Inter", sans-serif'
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
    letterSpacing: '0'
  micro-label:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Inter", sans-serif'
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.02em
  mono:
    fontFamily: '"SF Mono", "Menlo", "Fira Code", monospace'
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: '0'
rounded:
  sm: 0.25rem
  DEFAULT: 0.375rem
  md: 0.5rem
  lg: 0.625rem
  xl: 1rem
  full: 9999px
spacing:
  unit: 4px
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  gutter: 16px
  content-inset: 16px
---

## Brand & Style

Skillbox is a **native macOS desktop utility** — a local-first control center for managing AI agent skills across projects and providers. The visual language is clean and utilitarian: purposeful, fast-reading, and zero clutter. Every element earns its place by communicating state, enabling action, or providing context. No decorative flourishes, no marketing aesthetics.

The interface is a **fixed-window, panel-based desktop app** — compact system fonts at 13px, tight 4px-grid spacing, hairline borders as the primary separator, and a two-panel shell with a fixed sidebar. It is not a web app running in a browser frame. It is optimized for mouse precision and keyboard navigation, not touch or scroll. Administrative clarity is the priority over visual polish.

## Colors

The palette is anchored in **cool-neutral light surfaces**. The window background uses near-white Whisper Grey (`#F9FAFB`) to distinguish it from pure white content panels, creating a depth hierarchy through color temperature rather than shadows.

**Confident Red** (`#E11D48`) is the sole accent — primary buttons, active sidebar states, links, and focus rings. Its pale rose container (`#FFF1F2`) marks selected rows and info badges without tinting the neutral palette. Secondary grey (`#6B7280`) covers icons, inactive nav items, and secondary actions.

**Semantic states** are vivid against the neutral canvas: Caution Amber (`#F59E0B`) for warnings, Alert Red (`#EF4444`) for errors and broken symlinks, Growth Green (`#22C55E`) for active and up-to-date statuses. Each has a pale tinted container (`#FFFBEB`, `#FEF2F2`, `#F0FDF4`) for inline banners.

## Typography

The system uses the **macOS system font stack** (`-apple-system, BlinkMacSystemFont, "Inter"`) — SF Pro renders natively on macOS, Inter as a clean fallback. No custom font loading, no Google Fonts. The app feels native from the first render.

Base size is **13px** — the standard for compact desktop utilities. Smaller than web convention, appropriate for data-dense tables, metadata rows, and file path labels. Semibold (600) for headings provides hierarchy without heavy display weights. A monospace stack (SF Mono / Menlo) is used for every technical identifier: file paths, commit hashes, checksums — rendered at 12px on a subtle Frost Panel background to visually separate code values from prose labels.

## Layout & Spacing

The app occupies a **fixed-size desktop window** — no responsive layout, no breakpoints, no scrollable page. The window has a standard macOS title bar with traffic light buttons (close/minimize/zoom). The toolbar row sits directly below the title bar and holds global actions (Scan, Fetch, Open Skill Host Folder).

Below the toolbar, the layout is a **non-resizable two-panel shell**: a 220px fixed sidebar on the left and a flex-fill main content area on the right. The sidebar has Whisper Grey background with a 1px hairline right border. The content area uses a **16px inset** on all edges — tighter than a web page margin, consistent with native desktop panel padding.

Spacing follows a strict **4px grid**. Table cell padding is 8px vertical / 14px horizontal. Component internal padding is 12px. Section gaps between groups are 16px. The goal is information density: everything the user needs visible without scrolling whenever possible.

## Elevation & Depth

The interface is essentially **flat**. Depth is communicated through background color differences and hairline borders, not shadows. The sidebar (`#F9FAFB`) reads as recessed against the white content area (`#FFFFFF`) — no shadow needed.

The only two exceptions: **cards and info panels** use a single 1px `outline-variant` border (`#E4E4E7`) with no shadow. **Modals and confirmation dialogs** use a minimal ambient shadow (`0 2px 8px rgba(0,0,0,0.08)`) and a 1px border to lift them off the window without theatrical depth. No gradients, no blurred glass effects, no layered elevation.

## Shapes

Shape language is **restrained**. Buttons and inputs use a 6px radius (DEFAULT) — present but not soft. Cards and panels use 8px (md). Confirmation dialogs use 10px (lg). Status badges are pill-shaped (full / 9999px) for immediate recognition as status chips.

Tables, toolbars, and the sidebar shell have no border radius — they are flush, rectangular panels that connect to window edges, consistent with native desktop app conventions. Rounded corners are reserved for interactive components that stand alone within a panel.

## Components

### Toolbar

A slim horizontal bar (32px height) below the title bar with Whisper Grey background and a 1px hairline bottom border. Holds global actions as compact secondary buttons or icon buttons. Typography: 12px Medium. Separator lines between action groups.

### Buttons

Primary buttons: solid Confident Red (`#E11D48`) fill, white text, 6px radius, 6px vertical / 12px horizontal padding. Compact — sized for mouse precision, not touch. Secondary buttons: white fill, hairline border, Charcoal text, same shape. Ghost/icon buttons: no border or fill, Frost Panel on hover. Destructive actions appear only inside confirmation dialogs.

### Tables & Lists

Table headers: surface-container-high (`#E4E4E7`) fill, 11px Semibold uppercase in `on-surface-variant`. Data rows: white fill, hairline bottom separator. Row hover: surface-container-low (`#F4F4F5`). Selected row: primary-container (`#FFF1F2`) background with a 2px left solid Confident Red (`#E11D48`) accent border. File paths and hashes always use the monospace stack inside a subtle surface-container-low inline block.

### Status Badges

Pill-shaped chips (11px Medium, 3px vertical / 8px horizontal padding). Five states: Green on Pale Green (active / up-to-date), Amber on Pale Amber (warning / needs sync), Red on Pale Red (error / broken), Grey on Frost Panel (neutral / not configured), Blue on Soft Blue Fill (info / experimental).

### Navigation Sidebar

Nav items: 13px Regular, 6px vertical / 10px horizontal padding, 4px radius. Active item: primary-container (`#FFF1F2`) background, 2px left solid Confident Red (`#E11D48`) border accent. Inactive hover: surface-container-low. Section group labels: 11px Medium, Ghost Text (`#A1A1AA`), uppercase, non-interactive. Sidebar icons are allowed in Phase 1 when they improve scanability. Use small lucide icons only, keep them secondary in color, and never let icons dominate the label.

### Inline Warning & Error Banners

Full-width inline banners with 8px vertical / 14px horizontal padding. Warning: tertiary-container (`#FFFBEB`) fill, 3px solid tertiary (`#F59E0B`) left border. Error: error-container (`#FEF2F2`) fill, 3px solid error (`#EF4444`) left border. Always include a concise message and an inline action link where a recovery path exists.
