---
name: Precision Telematics
colors:
  surface: '#faf8ff'
  surface-dim: '#d2d9f4'
  surface-bright: '#faf8ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f2f3ff'
  surface-container: '#eaedff'
  surface-container-high: '#e2e7ff'
  surface-container-highest: '#dae2fd'
  on-surface: '#131b2e'
  on-surface-variant: '#434655'
  inverse-surface: '#283044'
  inverse-on-surface: '#eef0ff'
  outline: '#747686'
  outline-variant: '#c4c5d7'
  surface-tint: '#2151da'
  primary: '#0037b0'
  on-primary: '#ffffff'
  primary-container: '#1d4ed8'
  on-primary-container: '#cad3ff'
  inverse-primary: '#b7c4ff'
  secondary: '#006a61'
  on-secondary: '#ffffff'
  secondary-container: '#86f2e4'
  on-secondary-container: '#006f66'
  tertiary: '#004870'
  on-tertiary: '#ffffff'
  tertiary-container: '#006194'
  on-tertiary-container: '#b2d9ff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dce1ff'
  primary-fixed-dim: '#b7c4ff'
  on-primary-fixed: '#001551'
  on-primary-fixed-variant: '#0039b5'
  secondary-fixed: '#89f5e7'
  secondary-fixed-dim: '#6bd8cb'
  on-secondary-fixed: '#00201d'
  on-secondary-fixed-variant: '#005049'
  tertiary-fixed: '#cce5ff'
  tertiary-fixed-dim: '#93ccff'
  on-tertiary-fixed: '#001d31'
  on-tertiary-fixed-variant: '#004b73'
  background: '#faf8ff'
  on-background: '#131b2e'
  surface-variant: '#dae2fd'
typography:
  display-lg:
    fontFamily: Inter
    fontSize: 36px
    fontWeight: '700'
    lineHeight: 44px
    letterSpacing: -0.025em
  display-lg-mobile:
    fontFamily: Inter
    fontSize: 28px
    fontWeight: '700'
    lineHeight: 36px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Inter
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Inter
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.015em
  headline-sm:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.01em
  body-lg:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
    letterSpacing: -0.005em
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: 0em
  body-sm:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
    letterSpacing: 0.005em
  label-lg:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: 0.01em
  label-md:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.015em
  label-sm:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '600'
    lineHeight: 14px
    letterSpacing: 0.04em
  data-mono:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: -0.01em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1rem
  gutter-mobile: 0.75rem
  gutter-desktop: 1.5rem
  margin: 1.5rem
  margin-mobile: 1rem
  margin-desktop: 2rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2rem
---

## Brand & Style

This design system embodies **Precision Industrial SaaS**: an authoritative, ultra-crisp interface engineered for real-time fleet operations, spatial route tracking, and high-density telemetry observability. 

Targeting fleet dispatchers, logistics operations leads, and vehicle engineers, the UI evokes:
- **Absolute Operational Control:** Zero visual clutter, tight data alignment, and immediate recognition of critical vehicle metrics.
- **Instrument-Grade Precision:** Sharp lines, technical data readouts, micro-status indicators, and structural clarity over ornamental flair.
- **Modern Enterprise Rigor:** Balanced typography and purposeful, cool-slate tonal hierarchy that prevents cognitive fatigue across multi-monitor control centers and 12-hour operational shifts.

## Colors

The palette balances deep authoritative navigational primaries with high-visibility telemetry status accents, mapped over an ultra-clean, cool light slate background.

### Surface and Canvas
- **Canvas Base:** `#F8FAFC` (Slate 50) creates a bright, glare-reduced working backdrop.
- **Card & Panel Surface:** `#FFFFFF` (Pure White) houses active content, modular widgets, and data grids.
- **Inset / Muted Surface:** `#F1F5F9` (Slate 100) provides structure for data table headers, secondary sidebars, telemetry metric chips, and nested panels.

### Structural Borders
- **Standard Border:** `#E2E8F0` (Slate 200) defines crisp card perimeters, input edges, and divider lines.
- **Subtle Emphasis Border:** `#CBD5E1` (Slate 300) highlights focused controls, column dividers, and active toolbars.

### Brand & Telemetry Accents
- **Primary Accent (`#1D4ED8`):** Deep electric blue for primary interactive triggers, active breadcrumbs, route vectors, and selected states.
- **Telemetry Secondary (`#0D9488`):** Deep technical teal for engine health, geofence compliance, fuel optimization, and positive hardware pings.
- **Signal Tertiary (`#0284C7`):** Precision sky-cyan for streaming telemetry bursts, active GPS signal strength, and live motion state.

### Neutral Typography
- **Primary Slate (`#0F172A`):** High-contrast header and primary numerical data.
- **Secondary Slate (`#475569`):** Readable body descriptions, column labels, and metadata.
- **Muted Slate (`#64748B`):** Timestamps, sensor units, secondary telemetry indicators, and inactive tab labels.

## Typography

Typography relies entirely on **Inter** with calibrated letter spacing to maximize tabular legibility and rapid eye scanning.

- **Tabular Numerics:** Enable OpenType tabular figures (`tnum`) for all numerical indicators, speed metrics, coordinates, and real-time odometer counters to eliminate layout jitter during high-frequency live polling.
- **Optical Tightening:** Display and headline sizes incorporate micro-negative tracking (`-0.015em` to `-0.025em`) for a solid architectural lockup.
- **Labels & Micro-data:** `label-sm` utilizes uppercase casing with `0.04em` tracking for sensor badges (e.g., `VIN`, `RPM`, `CAN-BUS`, `DEF-LVL`).

## Layout & Spacing

The layout is built upon an **industrial fluid-grid** designed to transition seamlessly between map canvas split-screens, multi-column fleet telemetry rosters, and dense dashboard analytics.

### Breakpoints & Viewport Rules
- **Desktop (1440px+):** 12-column layout with `margin-desktop` (2rem / 32px) and `gutter-desktop` (1.5rem / 24px). Persistent left utility rail (64px collapsed, 240px expanded) paired with dynamic split-pane spatial map canvases.
- **Tablet (768px – 1439px):** 8-column layout with `gutter` (1rem / 16px). Secondary telemetry panels collapse into sliding bottom drawer sheets or switch to stacked mode.
- **Mobile (320px – 767px):** 4-column layout with `margin-mobile` (1rem / 16px) and `gutter-mobile` (0.75rem / 12px). Full-bleed map overlays with sheet interactions for individual vehicle telemetry inspection.

### Density Rhythm
Spacings rely on a strict 4px/8px incremental rhythm. In-component data grids prioritize compact vertical padding (`space-xs` to `space-sm`) to maximize visible asset rows per screen.

## Elevation & Depth

This system avoids heavy drop shadows, instead using **crisp structural borders paired with micro-ambient elevations** to maintain an authentic industrial hardware feel.

- **Level 0 (Base Canvas):** `#F8FAFC`. Zero elevation, non-interactive foundation.
- **Level 1 (Card / Table Layer):** `#FFFFFF` bordered with `1px solid #E2E8F0`, accompanied by a micro-shadow: `0 1px 2px 0 rgba(15, 23, 42, 0.04)`.
- **Level 2 (Hovered Card / Floating Toolbars):** `#FFFFFF` bordered with `1px solid #CBD5E1`, with shadow: `0 4px 6px -1px rgba(15, 23, 42, 0.06), 0 2px 4px -2px rgba(15, 23, 42, 0.04)`.
- **Level 3 (Flyouts / Context Menus / Telemetry Overlays):** `#FFFFFF` with shadow: `0 10px 15px -3px rgba(15, 23, 42, 0.08), 0 4px 6px -4px rgba(15, 23, 42, 0.03)`.
- **Active Focus Rings:** `0 0 0 2px #FFFFFF, 0 0 0 4px #1D4ED8` providing distinct high-contrast keyboard navigation.

## Shapes

The interface employs a disciplined **Soft (4px to 8px)** curvature logic, preserving structural rectangular alignment while easing visual sharpness:

- **Inputs, Buttons, and Chips:** Strict `4px` (`rounded-sm`) radius, creating a solid, calibrated appearance like physical dashboard controls.
- **Data Cards and Container Panels:** `6px` to `8px` (`rounded-md` / `rounded-lg`) radius to clearly group modules without wasting corner canvas area.
- **Pills / Status Dots:** Fully rounded (`rounded-full`) reserved strictly for live state indicators (pulsing telemetry beacons, mini-counters, and status badges).

## Components

### Buttons
- **Primary:** Background `#1D4ED8`, text `#FFFFFF`, border `1px solid transparent`, radius `4px`. Hover: `#1E40AF`. Active: `#1E3A8A`.
- **Secondary (Telemetry Action):** Background `#FFFFFF`, text `#0F172A`, border `1px solid #E2E8F0`. Hover: background `#F1F5F9`, border `#CBD5E1`.
- **Ghost / Tool:** Transparent background, text `#475569`. Hover: background `#F1F5F9`, text `#0F172A`.

### Status Chips & Badges
- **Active / Moving:** Background `#CCFBF1`, text `#0F766E`, border `1px solid #99F6E4`. Left icon or `6px` solid dot in `#0D9488`.
- **Streaming Telemetry:** Background `#E0F2FE`, text `#0369A1`, border `1px solid #BAE6FD`.
- **Critical Alert / Idle:** Background `#FEE2E2`, text `#B91C1C`, border `1px solid #FECACA`.

### Data Tables & Roster Lists
- **Header:** `#F1F5F9` background, `1px solid #E2E8F0` bottom edge, `label-md` uppercase text in `#475569`.
- **Row:** Height `44px` (dense) to `52px` (standard), bottom border `1px solid #F1F5F9`. Hover: `#F8FAFC`.
- **Numeric Cells:** Formatted via `data-mono` with right-alignment and tabular spacing.

### Input Fields & Controls
- **Inputs:** `#FFFFFF` background, `1px solid #E2E8F0` border, `4px` radius, text `#0F172A`. Placeholder: `#94A3B8`. Focus: border `#1D4ED8` with `0 0 0 1px #1D4ED8`.
- **Checkboxes & Radios:** `16px` squared with `3px` radius. Active state filled with `#1D4ED8` and white check icon.

### Cards & Telemetry Widgets
- Multi-tier surface structure: card container `#FFFFFF`, header bar with `#E2E8F0` bottom separator, metric values displayed in `display-lg` or `headline-lg` in `#0F172A`, accompanied by a unit indicator in `body-sm` `#64748B`.

### Telemetry-Specific Components
- **Vehicle HUD Drawer:** Floating Level 2 card docked above the map canvas showcasing real-time GPS coordinate lock, current speed, fuel/battery level, and driver status.
- **Route Breadcrumb Vector:** `#1D4ED8` active route line with `#0284C7` historic breadcrumb trail and directional arrowheads.