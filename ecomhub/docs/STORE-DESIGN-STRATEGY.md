# EcomHub Theme Engine Strategy

## 1. Vision
EcomHub storefronts should deliver:
**Designer-level results with near-zero effort.**

We achieve this by shifting from:
**Manual styling → Brand DNA system**

### Core Principle
`Soul (brand intent) → DNA (tokens) → Shape (UI)`

---

## 2. Product Concept: “Bitmoji for Stores”
Instead of editing CSS, merchants:
1. **Pick a brand personality** (Preset)
2. **Adjust a few controlled sliders** (Tokens)
3. **Get a high-quality store instantly** (Preview)

---

## 3. Design Philosophy
### Pillars
- **Curated, not infinite**: Limit options to ensure quality.
- **Composable pieces**: Small tokens combine for massive variety.
- **Instant visual feedback**: See changes as they happen.
- **Always professional output**: The system prevents "ugly" designs.

---

## 4. System Architecture

### 4.1 StoreTheme (DNA Model)
```go
type StoreTheme struct {
    // Base identity
    Preset        string  `json:"preset"`        // minimal | luxury | playful | tech

    // Sliders (bounded)
    Rounding      float64 `json:"rounding"`      // 0.0 → 1.0
    SpacingScale  string  `json:"spacing_scale"` // compact | normal | airy

    // Components
    FontPair      string  `json:"font_pair"`     // modern | elegant | bold
    SurfaceStyle  string  `json:"surface_style"` // solid | glass | bordered
    ButtonStyle   string  `json:"button_style"`  // square | rounded | pill

    // Colors (optional override)
    PrimaryColor  string  `json:"primary_color"`
    AccentColor   string  `json:"accent_color"`

    // Versioning (future-proof)
    Version       int     `json:"version"`
}
```

### 4.2 Rendering Model
**Template Integration**
```html
<body 
  class="preset-{{.Theme.Preset}} surface-{{.Theme.SurfaceStyle}} font-{{.Theme.FontPair}} spacing-{{.Theme.SpacingScale}}"
  style="--rounding: {{.Theme.Rounding}}">
```

**CSS Implementation**
```css
:root {
  --radius: calc(var(--rounding) * 20px);
}

.card {
  border-radius: var(--radius);
}

/* Preset Example */
.preset-luxury {
  --primary: #111111;
  --accent: #c9a96e;
}
```

---

## 5. Guardrails
To prevent poor designs:
- **Clamp all numeric values**: Prevent extreme values that break the UI.
- **Restrict enums**: Use predefined options only.
- **Semantic tokens**: No raw CSS injection allowed.
- **Contrast checks**: Always ensure color readability.

---

## 6. Roadmap

### Phase 1 — Prototype (The Foundation)
**Goal**: Validate concept with a minimal system.
- **Features**:
  - Preset selection: `minimal`, `luxury`, `playful`.
  - Rounding slider (0.0 → 1.0).
- **Deliverables**:
  - Extend `StoreTheme` model.
  - Update CSS variable engine.
- **Success Criteria**: Store appearance changes instantly with zero breakage.

### Phase 2 — MVP (The Creator)
**Goal**: Enable real merchant customization.
- **Features**: `FontPair`, `SurfaceStyle`, `SpacingScale`, `ButtonStyle`.
- **UI**: 4-step dashboard (Style → Feel → Preview → Save).
- **Deliverables**: Full validation pipeline and JSONB persistence.

### Phase 3 — AI Layer (“Magic Wand”) [DEFERRED]
**Goal**: Make customization feel effortless (to be implemented after Phase 2 is stable).
- **Input**: User prompt (e.g., "What do you sell?").
- **Output**: Valid `StoreTheme` JSON (Preset, Colors, Tokens).
- **Rule**: AI outputs DNA data, NOT CSS.
- **Note**: This phase is kept for future development to focus on a robust manual foundation first.

### ✅ Safety & Constraints

### 1. Protection by Defaults
Every design input is validated and clamped to safe ranges.
- **Rounding**: Clamped between `0.0` and `1.0`.
- **Presets**: Validated against allowed values (`minimal`, `luxury`, `playful`).
- **Automatic Repair**: Invalid or missing values are automatically repaired to sensible defaults.

### 2. Scoped Styling
All theme styles are isolated within the `.storefront` class.
- Prevents "Theme Leakage" into the dashboard or checkout pages.
- Allows for easy extension of global platform styles without breaking individual stores.

---

## 🕰️ Backward Compatibility

Existing stores are protected by defaults and normalization:

1. **Defaults**: Stores without a configuration automatically inherit the "Minimal" preset with medium rounding.
2. **Sanitization**: Old configuration data is automatically repaired upon first load/save.
3. **Variable Bridge**: Legacy CSS variables are mapped to new DNA tokens to ensure consistent rendering.

---

## 🚀 Roadmap:
- **Gallery**: Browse community and official presets.
- **Industry Templates**: Fashion (Luxury), Tech (Minimal), Food (Artisan).
- **A/B Testing**: Test which theme converts better.

---

## 7. Scalability Strategy
- **Technical**: CSS variables ensure low runtime cost and high cacheability.
- **Product**: Versioned themes allow for safe engine updates without breaking old stores.

---

## 8. Metrics
- **Conversion Rate**: Do certain presets drive more sales?
- **Efficiency**: Time to first store publish.
- **Engagement**: Percentage of users who utilize the "Magic Wand" vs. manual tweaks.

---

## 9. Risks & Mitigations
| Risk | Mitigation |
| :--- | :--- |
| **Ugly Stores** | Clamp values + Curated presets |
| **Over-complex UI** | Tabbed interface + Limited options |
| **AI Inconsistency** | Strict JSON schema validation |

---

## 11. Core System Principles
To maintain the integrity of the design system, all development must adhere to these rules:
- **Token-Derived UI**: All UI styles must be derived from `StoreTheme` tokens.
- **No Direct CSS Injection**: Merchants (and AI) cannot inject raw CSS strings.
- **Strict Validation**: All inputs must be validated and normalized (clamped) before storage and rendering.
- **Class-Based Rendering**: Use CSS classes and variables; avoid complex Go template logic for styling.
- **AI Constraints**: AI must output only valid `StoreTheme` JSON matching the schema.

---

## 12. Guiding Principle
**Don’t let users design from scratch.**  
**Let them assemble a beautiful brand.**
