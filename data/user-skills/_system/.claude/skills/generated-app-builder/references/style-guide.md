# Func-Operation Generated App Style Guide

Use this style guide for every generated app under `generated_apps/<uuid>/`. The goal is a consistent, quiet operational UI that feels like one product surface even when features are generated independently.

## Visual Direction

Generated apps are internal business tools. Prefer dense, readable, work-focused interfaces over decorative pages.

Use:

- White and near-white surfaces.
- Neutral gray text and borders.
- Blue primary actions.
- Green, amber, red, and gray status colors.
- 6px to 8px border radius.
- Compact spacing and table/list-first layouts.

Avoid:

- Marketing hero pages.
- Large decorative gradients.
- Purple-heavy palettes.
- Card-only dashboards with no workflow.
- Oversized typography.
- Rounded pill-heavy UI for normal controls.
- Explanatory text replacing real controls.

## Design Tokens

Use these values in `frontend/styles.js`. It is fine to implement them as CSS variables scoped under the app root.

```css
--ga-font: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
--ga-text: #1f2937;
--ga-text-strong: #111827;
--ga-text-muted: #6b7280;
--ga-text-subtle: #9ca3af;
--ga-bg: #ffffff;
--ga-bg-soft: #f9fafb;
--ga-bg-muted: #f3f4f6;
--ga-border: #e5e7eb;
--ga-border-strong: #d1d5db;
--ga-primary: #1677ff;
--ga-primary-hover: #4096ff;
--ga-primary-soft: #f0f5ff;
--ga-success-bg: #ecfdf5;
--ga-success: #059669;
--ga-warning-bg: #fffbeb;
--ga-warning: #d97706;
--ga-danger-bg: #fef2f2;
--ga-danger: #dc2626;
--ga-info-bg: #eff6ff;
--ga-info: #2563eb;
--ga-radius: 6px;
--ga-radius-lg: 8px;
--ga-shadow-popover: 0 18px 45px rgba(15, 23, 42, 0.16);
```

## CSS Scope

Every app must scope styles under one app root class. Use an app-specific prefix for selectors to avoid cross-app leakage.

Example:

```html
<div class="ga-book-root">
  ...
</div>
```

Recommended class pattern:

```text
.<prefix>-root
.<prefix>-header
.<prefix>-toolbar
.<prefix>-btn
.<prefix>-btn-primary
.<prefix>-table
.<prefix>-badge
.<prefix>-modal
.<prefix>-toast
```

Do not style generic selectors such as `button`, `input`, `table`, `h2`, or `.modal` without the app root prefix.

## Standard Layout

Most generated apps should use this structure:

```text
root
  header
    title and short description
    primary action
  toolbar
    search input
    filters
    clear/reset action
  content
    table/list/cards
    empty/loading/error state
  optional detail region or modal
  toast region
```

Header:

- Title font size: 20px to 22px.
- Description font size: 13px to 14px.
- Primary action on the right for desktop.
- Stack title and actions on narrow screens.

Toolbar:

- Use flex with wrapping.
- Search input min width around 200px.
- Filters use select controls.
- Clear action is secondary.

Content:

- Tables are preferred for record management.
- Cards are acceptable when each record has several descriptive fields or status actions.
- Keep row height compact, with 10px to 14px vertical padding.

## Components

### Buttons

Base button:

- Inline-flex.
- 8px 14px or 8px 16px padding.
- 14px font size.
- 6px radius.
- 1px border.
- Disabled state must reduce opacity and block pointer interactions.

Variants:

- Primary: blue background, white text.
- Secondary: white background, gray border.
- Danger: red text, soft red hover background.
- Small: 4px 10px padding and 13px font size.

### Inputs And Selects

Use consistent input styling:

- 8px 12px padding.
- 1px gray border.
- 6px radius.
- 14px font size.
- Blue focus border with subtle focus ring.

Validation:

- Mark invalid fields with red border.
- Show a short field-level error near the field.
- Do not rely only on toast for validation errors.

### Tables

Table wrapper:

- 1px border.
- 8px radius.
- Horizontal overflow for narrow screens.

Table:

- Header background `#f9fafb`.
- Header text muted, 13px, 600 weight.
- Body text 14px.
- Row hover `#f9fafb` or `#f0f5ff` for clickable rows.
- Actions aligned right and kept on one line.

### Badges

Use small status badges:

- 2px 8px padding.
- 12px font size.
- 500 weight.
- 10px radius.

Status colors:

- Success/active: green.
- Warning/pending: amber.
- Danger/disabled: red.
- Neutral/done/archived: gray.
- Info/in-progress: blue.

### Modals

Use modals for create/edit/confirm flows.

Modal requirements:

- Overlay uses `position: absolute` and `inset: 0`, scoped to the app mount/root container.
- Do not use `position: fixed` for modal masks, drawers, or toasts inside generated apps embedded in the host shell.
- Centered panel.
- Width 420px to 560px, max-width `calc(100vw - 32px)`.
- White panel, 8px radius, popover shadow.
- Header, body, footer sections.
- Footer actions right aligned on desktop, full-width or wrapped on mobile.

Append modal nodes under the app root when CSS is scoped under the app root.

The app root must fill the host mount container:

- `width: 100%`
- `min-height: 100%`
- `box-sizing: border-box`

### Toasts And Alerts

Use toast for transient success/failure after actions. Use inline alert for persistent page-level errors.

Toast:

- Fixed or root-scoped top-right position.
- White or soft status background.
- 8px radius.
- 13px to 14px text.
- Auto-dismiss only for success; keep errors until user action when possible.

## Empty, Loading, And Error States

Every generated app must include:

- Loading state while fetching initial data.
- Empty state when no records exist.
- Empty search state when filters produce no results.
- Page-level error state when backend load fails.
- Form validation state before save.

Empty state should include a next action:

- No records: show "新建".
- No search results: show "清除筛选".
- Load failed: show "重试".

## Responsive Rules

At width below 720px:

- Header stacks vertically.
- Toolbar controls stretch to full width.
- Tables keep horizontal scroll instead of squeezing text.
- Row action buttons can wrap.
- Modal width uses `calc(100vw - 32px)`.
- Avoid fixed heights that clip content.

Do not scale font size with viewport width. Keep text readable and prevent overlap with wrapping and min/max widths.

## Frontend Style Module Contract

`frontend/styles.js` must export this named function:

```js
export function injectStyles() {
  if (document.getElementById('<prefix>-styles')) return
  const style = document.createElement('style')
  style.id = '<prefix>-styles'
  style.textContent = getStyles()
  document.head.appendChild(style)
}
```

Generated CSS should include all classes used by:

- `frontend.js`
- `frontend/ui.js`
- `frontend/modal.js`
- Toasts, overlays, dropdowns, and dynamically appended nodes

Before completion, compare rendered class names with CSS selectors. Missing modal/toast styles are a common failure.
