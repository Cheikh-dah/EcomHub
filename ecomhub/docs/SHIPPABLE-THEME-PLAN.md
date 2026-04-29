# EcomHub Theme Engine — Shippable Plan

## 🎯 Core Idea
```text
Soul (intent) → DNA (theme JSON) → Shape (UI)
```
We are not just building a theme editor; we are building a system that generates good design automatically.

---

## 🧩 Phase 1 — Prototype (1–2 days)
**Goal**: One click → store feels different.

### Model
```go
type StoreTheme struct {
    PrimaryColor string `json:"primary_color"`
    AccentColor  string `json:"accent_color"`

    Preset   string  `json:"preset"`   // minimal | luxury | playful
    Rounding float64 `json:"rounding"` // 0 → 1
    Version  int     `json:"version"`
}
```

### Normalize
```go
func (t *StoreTheme) Normalize() {
    if t.Preset != "minimal" && t.Preset != "luxury" && t.Preset != "playful" {
        t.Preset = "minimal"
    }

    if t.Rounding < 0 {
        t.Rounding = 0
    }
    if t.Rounding > 1 {
        t.Rounding = 1
    }

    if t.Version == 0 {
        t.Version = 1
    }
}
```

### Template
```html
<body class="preset-{{.Theme.Preset}}" style="--rounding: {{.Theme.Rounding}}">
```

### CSS
```css
:root {
  --radius: calc(var(--rounding) * 20px);
}

.card,
button {
  border-radius: var(--radius);
}

.preset-minimal {
  --primary-color: #111;
  --accent-color: #2563eb;
}

.preset-luxury {
  --primary-color: #111;
  --accent-color: #c9a96e;
}

.preset-playful {
  --primary-color: #7c3aed;
  --accent-color: #f97316;
}
```

**✅ Done when**: Switch preset → store feels like a different brand.

---

## 🧩 Phase 2 — MVP (3–5 days)
**Goal**: Give power without losing control.

### Add fields
```go
FontPair     string
SpacingScale string
SurfaceStyle string
ButtonStyle  string
```

### Allowed values
- **FontPair**: `modern` | `elegant` | `bold`
- **Spacing**: `compact` | `normal` | `airy`
- **Surface**: `solid` | `glass` | `bordered`
- **Button**: `square` | `rounded` | `pill`

### Template
```html
<body 
  class="
    preset-{{.Theme.Preset}}
    font-{{.Theme.FontPair}}
    spacing-{{.Theme.SpacingScale}}
    surface-{{.Theme.SurfaceStyle}}
    btn-{{.Theme.ButtonStyle}}
  "
  style="--rounding: {{.Theme.Rounding}}">
```

**✅ Done when**: User tweaks store and it ALWAYS looks good.

---

## 🧩 Phase 3 — AI (After stability)
**Goal**: Make design instant.
- **Input**: “What do you sell?”
- **Output**: JSON containing preset, font_pair, spacing_scale, etc.
- **Rule**: AI selects settings → system renders. AI never writes CSS.

---

## 🧩 Phase 4 — Platform Expansion
- Theme gallery
- Industry presets
- A/B testing
- Marketplace

---

## 🛡️ System Rules (Never break these)
- All UI comes from tokens.
- No raw CSS input.
- All values validated.
- Use classes, not template logic.
- AI outputs only valid JSON.

---

## ⚠️ What to avoid
- Too many options.
- "Free design" (The Wix problem).
- Raw CSS editor.
- Building AI too early.

---

## 🧠 The Flow
```text
User → Chooses preset
        ↓
StoreTheme JSON
        ↓
Normalize()
        ↓
CSS classes + variables
        ↓
Rendered UI
```

---

## 🔥 One line to remember
**Ship small → feel magic → expand.**
