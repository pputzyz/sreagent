# Product

## Register

product

## Users

SRE engineers and DevOps teams managing production alerting and incident response. Primary context: on-call shifts, often at night or under stress (3am incident triage). They need fast readability, minimal cognitive load, and confidence that the system is working. Secondary users: team leads configuring alert rules and escalation policies, and managers reviewing dashboards and post-mortems.

## Product Purpose

SREAgent is a self-hosted alert management and on-call response platform. It ingests alerts from Prometheus, VictoriaMetrics, and Zabbix, processes them through a normalization/deduplication/routing pipeline, and dispatches notifications via Lark (Feishu), email, or webhooks. It replaces ad-hoc alert scripts and spreadsheets with a structured incident workflow: alert → incident → channel → post-mortem.

Success: SRE teams trust it to wake them up reliably, reduce alert noise, and speed up incident resolution.

## Brand Personality

**Clean, warm, trustworthy.**

Three words: precise, approachable, dependable.

The interface should feel like a well-maintained control room: organized, calm, nothing wasted. Not clinical or cold (that's corporate SaaS), not playful or cute (that's consumer app). Warm in the way a trusted colleague is warm: competent, clear, doesn't waste your time.

Emotional goals:
- Confidence: "I trust this system to surface what matters"
- Calm: "I can think clearly even during an incident"
- Efficiency: "I can do what I need in 2 clicks, not 8"

## Anti-references

- **AI-generated aesthetics**: Rainbow gradients, glassmorphism, claymorphism, bouncy spring animations, aurora backgrounds, gradient text. These telegraph "template" not "tool".
- **Dark OLED monotone**: All-black themes with single accent color. Looks dead, not professional.
- **Corporate SaaS template**: Big hero metrics, identical card grids, modal-for-everything, generic "dashboard" look.
- **Data-dense terminal**: Pure function over form. Works for CLI tools, not for a platform people stare at 8+ hours.

Specific product anti-references:
- Grafana's default theme (too dark, too dense, no warmth)
- PagerDuty's UI (corporate SaaS template)
- Generic admin dashboard templates (Bootstrap/Element UI defaults)

## Design Principles

1. **Clarity under stress**: Every element earns its place. When an SRE is half-awake at 3am, the UI should guide their eyes to what matters. No decorative elements that compete with data.

2. **Warm neutrality**: The base palette is neutral (not cold blue-gray, not warm brown). Color is used sparingly and purposefully: severity indicators, status badges, section differentiation. The default state should feel calm, not monochrome.

3. **Progressive disclosure**: Show the essential first. Complexity appears when requested. Don't dump every option on screen. Sidebar groups collapse. Advanced filters hide behind toggles. Empty states explain what to do next.

4. **Consistent density**: Match information density to the task. Overview pages breathe. Detail pages compress. List views prioritize scannability. Forms prioritize clarity. Don't use the same spacing everywhere.

5. **System-first theming**: Both light and dark themes are first-class citizens. Neither is an afterthought. The system detects `prefers-color-scheme` and applies the appropriate theme. Both themes maintain the same warmth and readability.

## Accessibility & Inclusion

- WCAG 2.1 AA compliance target (minimum 4.5:1 contrast for text)
- Full keyboard navigation for all interactive elements
- `prefers-reduced-motion` respected for all animations
- Screen reader support via proper ARIA labels on interactive elements
- Touch targets minimum 44x44px
- Both light and dark themes must be independently usable (not just color-inverted)
