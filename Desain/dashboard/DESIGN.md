---
name: PasTrack Fleet Telematics
colors:
  surface: '#f7f9fb'
  surface-dim: '#d8dadc'
  surface-bright: '#f7f9fb'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f2f4f6'
  surface-container: '#eceef0'
  surface-container-high: '#e6e8ea'
  surface-container-highest: '#e0e3e5'
  on-surface: '#191c1e'
  on-surface-variant: '#434655'
  inverse-surface: '#2d3133'
  inverse-on-surface: '#eff1f3'
  outline: '#737686'
  outline-variant: '#c3c6d7'
  surface-tint: '#0053db'
  primary: '#004ac6'
  on-primary: '#ffffff'
  primary-container: '#2563eb'
  on-primary-container: '#eeefff'
  inverse-primary: '#b4c5ff'
  secondary: '#006398'
  on-secondary: '#ffffff'
  secondary-container: '#5bb8fe'
  on-secondary-container: '#00476e'
  tertiary: '#4b566a'
  on-tertiary: '#ffffff'
  tertiary-container: '#636e83'
  on-tertiary-container: '#ecf1ff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dbe1ff'
  primary-fixed-dim: '#b4c5ff'
  on-primary-fixed: '#00174b'
  on-primary-fixed-variant: '#003ea8'
  secondary-fixed: '#cce5ff'
  secondary-fixed-dim: '#93ccff'
  on-secondary-fixed: '#001d31'
  on-secondary-fixed-variant: '#004b73'
  tertiary-fixed: '#d8e3fb'
  tertiary-fixed-dim: '#bcc7de'
  on-tertiary-fixed: '#111c2d'
  on-tertiary-fixed-variant: '#3c475a'
  background: '#f7f9fb'
  on-background: '#191c1e'
  surface-variant: '#e0e3e5'
typography:
  display:
    fontFamily: Inter
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 40px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Inter
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.015em
  headline-lg-mobile:
    fontFamily: Inter
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.01em
  headline-md:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.01em
  title-sm:
    fontFamily: Inter
    fontSize: 15px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: -0.005em
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-sm:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
  data-tabular:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: -0.005em
  label-md:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.02em
  label-xs:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '600'
    lineHeight: 14px
    letterSpacing: 0.03em
  telemetry-metric:
    fontFamily: Inter
    fontSize: 22px
    fontWeight: '700'
    lineHeight: 26px
    letterSpacing: -0.02em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1rem
  gutter-dense: 0.5rem
  margin: 1.5rem
  margin-mobile: 0.75rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 0.75rem
  space-lg: 1rem
  space-xl: 1.5rem
---

## Brand & Style

This design system embodies high-precision industrial observability, mission-critical clarity, and enterprise discipline. Designed specifically for enterprise fleet dispatchers, logistics operations managers, and IoT sensor analysts, the interface emphasizes cognitive speed, zero-latency situational awareness, and effortless data scanning over visual decoration.

### Design Movement
**High-Density Technical Minimalism & Precision Instrumentation.** The aesthetic mirrors avionics flight displays and modern industrial SCADA systems translated into a refined SaaS environment:
- Crisp 1px structural dividing lines for unambiguous spatial boundaries.
- Ultra-clean light mode canvas with high-contrast foreground typography.
- Information hierarchy governed by color-coded telemetry status signals rather than heavy decorative styling.
- Compact, dense component footprints optimized for multi-screen monitoring centers and continuous operational use.

## Colors

The palette is anchored by a high-clarity slate foundation paired with electric core accents and dedicated semantic telematics tokens.

### Foundation & Structural Neutrals
- **Canvas Base:** `#F8FAFC` (Slate 50) serves as the ambient backdrop for application chrome, sidebars, and widget gutters.
- **Surface Elevation:** `#FFFFFF` (Pure White) provides maximum contrast for data cards, modal drawers, and telemetry tables.
- **Borders & Dividers:** `#E2E8F0` (Slate 200) for standard layout and card borders; `#CBD5E1` (Slate 300) for input frames and active dividers.
- **Typography Primary:** `#0F172A` (Slate 900) for headers, values, and high-priority readouts.
- **Typography Secondary:** `#475569` (Slate 600) for labels, units of measurement, and table column headers.
- **Typography Tertiary / Muted:** `#94A3B8` (Slate 400) for metadata timestamps, disabled states, and sensor breadcrumbs.

### Brand & Interaction Accents
- **Primary Action (Brand Cobalt):** `#2563EB` (Blue 600) for primary CTAs, active route polylines, and selected state rings.
- **Secondary Interaction (Precision Sky):** `#0284C7` (Sky 600) for telemetry toggles, geofence perimeters, and filter badges.
- **Control Bar Neutral (Deep Slate):** `#1E293B` (Slate 800) for primary command headers, dark tooltips, and map floating control docks.

### Operational Telematics & IoT Semantics
Status colors must never be used purely decoratively; they are functional signals:
- **Moving / Engine Active:** `#10B981` (Emerald 500) | Surface: `#ECFDF5` | Text: `#065F46`
- **Idle / Ignition On:** `#F59E0B` (Amber 500) | Surface: `#FFFBEB` | Text: `#92400E`
- **Parked / Signal Offline:** `#64748B` (Slate 500) | Surface: `#F1F5F9` | Text: `#334155`
- **Critical Alert / Fuel Drain & Theft:** `#EF4444` (Rose 500) | Surface: `#FEF2F2` | Text: `#991B1B`
- **Fuel Volume / Refuel Dynamic:** `#0EA5E9` (Sky 500) | Surface: `#F0F9FF` | Text: `#0369A1`

## Typography

This system strictly leverages **Inter** for all hierarchy layers, ensuring maximum legibility across dense coordinate streams, sensor charts, and hardware device registries.

### Typographic Principles
- **Tabular Figures:** Always apply `font-variant-numeric: tabular-nums` to all data cells, lat/long coordinates, odometer readings, fuel percentages, and timestamps to eliminate column jitter during live telemetry refreshes.
- **Metric Formatting:** Telemetry metrics use `telemetry-metric` (22px bold) with immediate micro-units (`label-xs` in uppercase `#64748B`, e.g., `L/100km`, `RPM`, `km/h`).
- **Uppercase Hierarchy:** Micro-labels, table headers, and status badges exclusively use uppercase styling (`label-xs` or `label-md`) with positive letter spacing (`0.02em` to `0.03em`) to anchor technical scan patterns.

## Layout & Spacing

The layout is built upon an adaptive, compact grid engine configured for live fleet orchestration.

### Grid System
- **Desktop (>= 1280px):** 12-column fluid grid with `1rem` (16px) gutters and `1.5rem` (24px) canvas margins. Monitoring split-views utilize a fixed `380px` collapsible telemetry asset list and a fluid canvas for map, geofence, and route layers.
- **Dense Telematics Mode:** When real-time multi-vehicle sensor tracking is active, the grid collapses to `gutter-dense` (`0.5rem` / 8px) to maximize dashboard widget packing.
- **Tablet (768px - 1279px):** 8-column layout. Sidebars collapse to icon-rail anchors (`64px`), widgets re-flow to 2-up spans.
- **Mobile (< 768px):** 4-column layout with `margin-mobile` (`0.75rem` / 12px). Full-screen bottom sheets replace persistent map side panels.

### Structural Flow Rules
- Telemetry widget inner padding is strictly locked to `space-md` (`0.75rem` / 12px) or `space-lg` (`1rem` / 16px).
- Vertical stack separation inside cards utilizes `space-sm` (`0.5rem` / 8px) between metric clusters.

## Elevation & Depth

This system avoids heavy, atmospheric shadows to maintain an ultra-crisp, surgical aesthetic. Depth is communicated primarily through **tonal separation** and **crisp 1px borders**, supported by restrained micro-shadows.

### Layer Architecture
- **Layer 0 (Canvas):** `#F8FAFC`. Background for non-interactive gutter surfaces.
- **Layer 1 (Card / Workspace):** `#FFFFFF`. Bound by a crisp `1px solid #E2E8F0` border. Zero shadow in neutral state; elevates slightly on hover with `box-shadow: 0 1px 3px 0 rgba(15, 23, 42, 0.05), 0 1px 2px -1px rgba(15, 23, 42, 0.05)`.
- **Layer 2 (Floating Map Controls / Filter Bars):** `#FFFFFF` surface with `1px solid #CBD5E1` and an anchor shadow: `box-shadow: 0 4px 6px -1px rgba(15, 23, 42, 0.08), 0 2px 4px -2px rgba(15, 23, 42, 0.04)`.
- **Layer 3 (Overlays, Modals, Slide-Over Diagnostics):** `#FFFFFF` surface with `1px solid #94A3B8` backed by an ambient drop: `box-shadow: 0 10px 15px -3px rgba(15, 23, 42, 0.1), 0 4px 6px -4px rgba(15, 23, 42, 0.05)`. Backdrop overlay is styled with `#0F172A` at `35%` opacity with a subtle `backdrop-filter: blur(2px)`.

### Border Discipline
All cards, tables, and metric containers must preserve crisp perimeter borders (`#E2E8F0`). Borders do not disappear when shadows are applied; they ensure precise alignment against map canvases and raster backgrounds.

## Shapes

The geometric framework favors controlled, engineered radii that retain high architectural rigidity.

- **Base Radius (0.25rem / 4px):** Applied to badges, inputs, buttons, table cell tag pills, and micro-metric chips.
- **Card & Container Radius (`rounded-lg`, 0.5rem / 8px):** Applied to telemetry cards, map control panels, and slide-in trays.
- **Inner Nested Radius:** When an inner component is nested inside an `8px` container with `8px` padding, its radius must scale down to `4px` to maintain concentric geometry.
- **Pill Exceptions:** Rounded-full (`9999px`) is reserved exclusively for vehicle status indicators (live pulsing dot circles) and fuel tank level fill bars.

## Components

### Buttons
- **Primary:** Background `#2563EB`, text `#FFFFFF`, 1px border `#1D4ED8`. Hover: `#1D4ED8`. Active: `#1E40AF`. Focus: `2px` ring `#93C5FD` with `2px` offset.
- **Secondary:** Background `#FFFFFF`, text `#1E293B`, 1px border `#E2E8F0`. Hover: `#F8FAFC`, border `#CBD5E1`.
- **Destructive / Fuel Kill Switch:** Background `#EF4444`, text `#FFFFFF`, border `#DC2626`. Hover: `#DC2626`. Focus ring `#FCA5A5`.
- **Dimensions:** Default height `36px` (`padding: 0 12px`, font: `13px 500`). Dense mode height `28px` (`padding: 0 8px`, font: `12px 500`).

### Telemetry Status Chips
- Rendered with `4px` radius, uppercase font (`11px 600`), and a leading 6px circular status dot.
- **Moving:** Dot `#10B981` (pulsing), background `#ECFDF5`, text `#065F46`, border `#A7F3D0`.
- **Idling:** Dot `#F59E0B`, background `#FFFBEB`, text `#92400E`, border `#FDE68A`.
- **Offline:** Dot `#64748B`, background `#F1F5F9`, text `#334155`, border `#E2E8F0`.
- **Alert / Theft:** Dot `#EF4444` (blinking), background `#FEF2F2`, text `#991B1B`, border `#FECACA`.

### Telematics Data Tables
- **Header:** Height `32px`, background `#F8FAFC`, text `#475569`, font: `11px 600` uppercase. 1px solid bottom border `#E2E8F0`.
- **Row:** Height `40px` (standard) or `32px` (dense). Alternating hover state: background `#F8FAFC`. Selected row: background `#EFF6FF` with `2px` left border in `#2563EB`.
- **Cell Alignment:** Numeric readouts (speed, fuel capacity, voltage, temperature) are right-aligned using tabular figures. Status chips and vehicle IDs are left-aligned.

### Input Fields & Search Filters
- Background `#FFFFFF`, height `36px` (or `28px` in toolbars). Border: `1px solid #CBD5E1`. Text: `#0F172A`, placeholder: `#94A3B8`.
- Focus state: Border `#2563EB`, outline none, box shadow: `0 0 0 1px #2563EB`.
- Leading/Trailing elements: Monochromatic fleet search icons and clear buttons rendered in `#64748B`.

### Cards & Telemetry Widgets
- Background `#FFFFFF`, border `1px solid #E2E8F0`, corner radius `8px`.
- **Widget Header:** Height `40px`, padding `0 12px`, separated by `1px solid #F1F5F9`. Features vehicle registration ID, current driver name, and operational status chip.
- **Widget Body:** Contains high-contrast metric pairs (label above value), fuel tank level gauge, and real-time sparkline telemetry.

### Fuel Telemetry Gauge (Specialized Component)
- **Bar Track:** Height `8px`, background `#E2E8F0`, radius `9999px`.
- **Fill Segment:** Dynamic width fill in `#0EA5E9` (Sky 500) under standard consumption; turns `#EF4444` instantly if sudden sensor drop gradient exceeds drain threshold (> 5L/min drop).
- **Metric Anchor:** Tabular fuel readout (e.g., `420.5 / 600 L • 70.1%`) directly paired below the gauge in `12px` font with secondary status text.