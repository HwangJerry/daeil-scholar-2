# Design System Guide For dflh-saf-v2

The web app and admin-facing UI work in this repository consume the workspace
design system from `../../../design-system`.

## Start Here

- Designer onboarding: `../../../design-system/docs/DESIGNER_ONBOARDING.md`
- Web engineering guide: `../../../design-system/docs/WEB_ENGINEERING_GUIDE.md`
- Token lifecycle: `../../../design-system/docs/TOKEN_LIFECYCLE.md`
- Contract lifecycle: `../../../design-system/docs/CONTRACT_LIFECYCLE.md`
- Validation workflow: `../../../design-system/docs/VALIDATION_WORKFLOW.md`

## Web Implementation Rules

- Use generated tokens from `../../../design-system/platform/web/design-tokens.css`.
- Keep `frontend/src/index.css` importing generated token CSS before Tailwind.
- Use token-backed classes for color, typography, spacing, radius, elevation,
  layout, iconography, motion, and state.
- Keep News Feed, Messages, and My Page aligned with `screen.feed`,
  `screen.messages`, and `screen.myPage`.
- Do not edit generated design-system artifacts manually.

## Required Gate

Run from the workspace root:

```bash
npm run verify-design-system
```

For visual changes, also follow
`../../../design-system/docs/VALIDATION_WORKFLOW.md`.
