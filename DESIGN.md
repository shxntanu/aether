---
name: Aether
description: A calm, private document vault rendered as a painted archive garden.
colors:
  mineral-paper: "#f7f4e9"
  archive-paper: "#fffdf4"
  warm-paper: "#eee8d9"
  blue-paper: "#dce7f1"
  deep-indigo: "#18385f"
  soft-indigo: "#31527d"
  faint-indigo: "#5d7090"
  archive-line: "#c4d1df"
  archive-line-strong: "#96afca"
  indigo: "#315e9d"
  indigo-deep: "#224b85"
  peach-folio: "#f2d7ca"
  moss: "#718c6d"
  archive-red: "#873f3b"
  roof-blue: "#c8d7ea"
  footer-paper: "#f5f0e4"
  ready-wash: "#e2ecd9"
  processing-wash: "#f6ebc8"
  attention-wash: "#f8e4dd"
  tag-wash: "#f0dfd3"
  tag-ink: "#6f3d31"
typography:
  display:
    fontFamily: "Newsreader Variable, Georgia, serif"
    fontSize: "clamp(32px, 4vw, 48px)"
    fontWeight: 560
    lineHeight: 0.98
    letterSpacing: "-0.045em"
  headline:
    fontFamily: "Newsreader Variable, Georgia, serif"
    fontSize: "38px"
    fontWeight: 560
    lineHeight: 0.98
    letterSpacing: "-0.06em"
  title:
    fontFamily: "Newsreader Variable, Georgia, serif"
    fontSize: "25px"
    fontWeight: 620
    lineHeight: 1.16
    letterSpacing: "-0.03em"
  body:
    fontFamily: "Geist Variable, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Geist Variable, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "10px"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "0.1em"
rounded:
  sm: "3px"
  md: "4px"
  lg: "6px"
  pill: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "30px"
components:
  button-primary:
    backgroundColor: "{colors.indigo}"
    textColor: "{colors.archive-paper}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 14px"
    height: "40px"
  button-quiet:
    backgroundColor: "rgba(255, 253, 244, 0.55)"
    textColor: "{colors.soft-indigo}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 14px"
    height: "40px"
  button-danger:
    backgroundColor: "{colors.attention-wash}"
    textColor: "{colors.archive-red}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 14px"
    height: "40px"
  search-field:
    backgroundColor: "rgba(255, 253, 244, 0.96)"
    textColor: "{colors.deep-indigo}"
    typography: "{typography.body}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "44px"
  archive-container:
    backgroundColor: "{colors.archive-paper}"
    textColor: "{colors.deep-indigo}"
    rounded: "{rounded.md}"
  archive-tag:
    backgroundColor: "{colors.tag-wash}"
    textColor: "{colors.tag-ink}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 7px"
    height: "20px"
  status-ready:
    backgroundColor: "{colors.ready-wash}"
    textColor: "#426334"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 7px"
    height: "20px"
  status-processing:
    backgroundColor: "{colors.processing-wash}"
    textColor: "#705414"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 7px"
    height: "20px"
---

# Design System: Aether

## Overview

**Creative North Star: "Celestial Archive Garden"**

Aether is a functional archive rendered as a calm painted garden: mineral paper carries the work surface, indigo architectural lines give the shell structure, peach folios mark handled documents, and moss states add a quiet living signal. Fine screen-printed texture and authored scenic plates bring atmosphere to the roof eave, navigation rail, and wave footer while the document workspace stays dense and legible.

The visual story follows the work: members enter a protected library, scan and filter preserved documents, open one folio without losing the ledger, and upload with honest progress. The interface uses two voices—editorial Newsreader for the archive's names and headings, and practical Geist for utility text, metadata, and controls. Surfaces remain tactile and paper-like; depth is earned through a restrained paper lift rather than glossy effects.

**Key Characteristics:**

- Mineral paper and screen-printed texture as the default material.
- Indigo roof eaves, rules, waves, focus states, and primary actions.
- Peach folios, stamped tags, and moss/amber/rose lifecycle washes.
- A full-height scenic rail and a right-anchored wave veil framing a dense worktable.
- Bounded motion with an explicit reduced-motion presentation.

## Colors

The palette is a quiet mineral paper field with architectural indigo, warm peach folios, cool blue paper, and moss-led state signals.

### Primary

- **Archive Indigo**: the structural accent for the roof eave, active navigation, primary buttons, selection edge, focus ring, and wave treatment.
- **Deep Indigo Ink**: the main reading color for titles, body copy, and the dark upload queue.

### Secondary

- **Peach Folio**: the handled-paper accent used for selected rows, tags, document treatments, and danger-state warmth.
- **Moss**: the authenticated and ready-state signal, supported by a pale ready wash.

### Tertiary

- **Roof Blue**: the cool architectural field behind the indigo command bar.
- **Blue Paper**: the cool preview and document surface that separates inspection from the mineral worktable.

### Neutral

- **Mineral Paper**: the global textured canvas.
- **Archive Paper**: the lifted folio and dialog surface.
- **Warm Paper**: table headers, quiet hover surfaces, and rail details.
- **Archive Line** and **Archive Line Strong**: fine rules, field borders, and separators.
- **Footer Paper**: the pale base of the wave footer.

### Named Rules

**The Painted Archive Rule.** Paper and scenic artwork establish atmosphere; the central ledger remains readable and operational.

**The Indigo Structure Rule.** Indigo is reserved for hierarchy, actions, active navigation, focus, and wave progress so it continues to read as architecture rather than decoration.

## Typography

**Display Font:** Newsreader Variable (with Georgia, serif)

**Body Font:** Geist Variable (with Inter, ui-sans-serif, system-ui, sans-serif)

**Label/Mono Font:** Geist Variable for labels; system monospace for original filenames.

**Character:** Newsreader gives the archive an editorial, handled-page voice. Geist keeps utility text, metadata, and controls crisp at the dense scale needed for a private worktable.

### Hierarchy

- **Display** (560, `clamp(32px, 4vw, 48px)`, 0.98): workspace and page titles.
- **Headline** (560, 38px, 0.98): signed-out, loading, and connection-state statements.
- **Title** (620, 25px, 1.16): inspector folio titles and dialog titles.
- **Body** (400, 16px, 1.5): default utility copy and supporting text.
- **Label** (700, 10px, 0.1em letter spacing): table headings, state labels, and small metadata, often uppercase.

### Named Rules

**The Two Voice Rule.** Use Newsreader for the archive's editorial layer; use Geist for controls, metadata, and operational status.

## Layout

The desktop worktable is a sticky command bar above a two-column shell: a scenic left rail and a fluid main workspace. The rail is `minmax(242px, 20vw)` at wide widths and 220px below 1080px; the main content is capped at 1220px with generous responsive padding. When a document is selected, the right inspector opens as a 440px-or-40vw folio sheet over the workspace.

At 1080px, the ledger removes the contributor/date column and the inspector narrows to `min(420px, 54vw)`. Between 781px and 900px, the ledger keeps only the document and action columns. At 780px and below, the rail becomes an off-canvas 244px panel, the workspace stacks its tools, the inspector becomes full-screen, member rows collapse to a single action, and the upload queue moves edge-to-edge with 12px insets. The top bar reduces to a 76px height on mobile; the wave veil reduces to 138px and softens so it stays atmospheric behind the workspace.

The desktop top bar uses a `clamp(92px, 9vw, 126px)` height. The post-header shell fills the viewport so the scenic rail reaches the bottom edge. A 190px wave veil sits outside document flow, starts at the workspace edge, anchors to the lower-right corner, and fades toward the rail without reserving a footer band. The shell keeps the document collection dense with fine rules, while previews, dialogs, and sheets receive their own paper surfaces.

## Elevation & Depth

This is a layered paper system with restrained lift. Flat tonal changes, fine rules, and the screen-printed plates do most of the depth work. The shared archive shadow (`0 18px 42px rgba(41, 73, 111, 0.15)`) is reserved for the inspector, mobile rail, popovers, and other lifted paper surfaces; row and button hover use smaller, local lifts. There are no glass surfaces or gradient washes in the implemented archive world.

### Shadow Vocabulary

- **Paper lift** (`0 18px 42px rgba(41, 73, 111, 0.15)`): inspector, mobile rail, select/account popovers, and lifted paper surfaces.
- **Handled folio lift** (`0 10px 24px rgba(41, 73, 111, 0.13)`): the selected document row.
- **Quiet control lift** (`0 5px 12px rgba(35, 71, 115, 0.15)`): button hover and similarly tactile controls.

### Named Rules

**The Paper Lift Rule.** Surfaces are calm at rest; shadows and a 2px upward translation appear only when a document or control is handled.

## Shapes

The archive uses small, gently softened corners rather than a rounded-card language. The base archive radius is 4px, with 3px controls and tags, 6px popovers and selectors, and a 999px search pill. Rules are one-pixel blue-gray lines; the upload dropzone is the only recurring dashed border. Circular marks and avatars are reserved for identity, not generic decoration.

## Components

### Buttons

- **Shape:** compact archive controls with 3px corners, 40px minimum height on desktop and 44px touch targets on mobile.
- **Primary:** Archive Indigo with Archive Paper text, 14px horizontal padding, and a small upload/add icon when action context benefits from it.
- **Hover / Focus:** hover deepens the indigo or paper surface, adds a restrained local shadow, and lifts 2px; all interactive controls receive a 3px indigo focus ring with a 3px offset.
- **Quiet / Danger / Icon:** quiet controls use a translucent archive-paper wash and line border; danger uses a rose wash and archive red; icon controls stay transparent until hovered.

### Chips

- **Style:** stamped peach tags use the tag wash, tag ink, 3px corners, 10px Geist labels, and 20px line height.
- **State:** lifecycle chips use pale ready/moss, processing/amber, or attention/rose washes with a small status dot.

### Cards / Containers

- **Corner Style:** collections and sheets use the 4px archive radius; individual grid cards and previews stay closer to square.
- **Background:** Archive Paper for collections, grids, sheets, dialogs, and the login folio; Mineral Paper and Blue Paper separate work and preview zones.
- **Shadow Strategy:** use the Paper Lift and Handled Folio Lift vocabulary in Elevation & Depth.
- **Border:** one-pixel Archive Line rules; selected rows add a 3px indigo inset edge.
- **Internal Padding:** 14–22px for dense rows and sheets; larger 42px breathing room on the login folio.

### Inputs / Fields

- **Style:** mineral-paper fields with a 1px strong line, 3px corners, deep indigo text, and 38px height; the global search is the exception, using a 44px pill.
- **Focus:** the shared visible indigo focus ring remains present for keyboard use; editable fields deepen their border on hover.
- **Error / Disabled:** errors use the rose notice treatment; disabled buttons retain their shape and become 50% opaque with no lift.

### Navigation

- **Style:** the scenic rail uses Newsreader labels, archive-paper nav items, 4px corners, and indigo linework. The current route becomes an indigo-and-wave item with archive-paper text.
- **Default / Hover / Active:** default items are pale and quiet, hover shifts 3px toward the content and strengthens the line, active items use the indigo wave treatment.
- **Mobile treatment:** navigation is an off-canvas 244px rail below 780px, with an inerted background, focus containment, Escape dismissal, and a scrim.

### Signature: Wave Upload Progress

The upload queue is a dark, bottom-anchored status surface with an honest sequential count and an accessible live progressbar. Its fill scales from the left through `--vault-progress`; an indeterminate upload holds at a partial fill while the indigo wave plate drifts horizontally. Completed files advance the queue, and failed files expose an explicit Retry action. On mobile the queue spans the viewport with 12px side insets.

## Do's and Don'ts

### Do:

- **Do** keep mineral paper, fine blue-gray rules, and the authored texture as the default work surface.
- **Do** use indigo for structure and action: roof eaves, active navigation, focus, selection, primary buttons, and wave progress.
- **Do** pair Newsreader archive headings with Geist utility and metadata text.
- **Do** preserve the full-height scenic rail and faded wave veil as framing elements around the dense worktable.
- **Do** keep motion bounded and honor `prefers-reduced-motion: reduce` by removing meaningful animation and transition duration.

### Don't:

- **Don't** introduce generic SaaS cards, glass surfaces, gradients, or ornamental pseudo-Asian typography.
- **Don't** use scenic artwork behind dense document rows where it would reduce legibility.
- **Don't** turn peach or moss into a general-purpose accent; they identify folios and lifecycle states.
- **Don't** hide upload state behind a spinner or animation; the queue must expose its count, status, and progress semantics.
- **Don't** remove keyboard focus, live-region announcements, or the mobile navigation's focus containment while restyling.
