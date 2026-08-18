# dflh-saf-v2 Web Frontend

React + TypeScript + Vite frontend for `dflh-saf-v2`.

## Design System Usage

The web frontend consumes the cross-platform design system from the workspace-level `design-system` package.

- Full onboarding and lifecycle docs live in `../../design-system/docs/`.
- Use `../../design-system/docs/WEB_ENGINEERING_GUIDE.md` as the platform guide.
- Use `../../design-system/docs/VALIDATION_WORKFLOW.md` before merging UI changes.
- Canonical tokens live in `../../design-system/tokens/design-tokens.json`.
- Generated web tokens live in `../../design-system/platform/web/design-tokens.css`.
- `src/index.css` imports the generated token CSS before Tailwind.
- Shared screen contracts live in `../../design-system/contracts/component-contracts.json`.

When adding or changing UI:

- Use Tailwind classes backed by generated DS variables such as `text-text-primary`, `bg-surface`, `border-border`, `shadow-card`, and radius/spacing utilities mapped from tokens.
- Do not introduce ad-hoc hex colors, one-off font stacks, or non-tokenized surface/radius choices in core screens.
- For News Feed, Messages, and My Page, keep the implementation aligned with the contracts `screen.feed`, `screen.messages`, and `screen.myPage`.
- If a web-only visual difference is intentional, document it in `../../design-system/verification/reports/accepted-deltas.json`.

## Visual Parity Build

Use the visual-check build when capturing design-system baselines. It bypasses the temporary WIP gate only for visual verification.

```bash
cd ../../../
npm run build-web-for-visual-check
cd dflh-saf-v2/frontend
npm run preview -- --host 127.0.0.1 --port 4173
```

Then from the workspace root:

```bash
npm run visual-check-web
```

For baseline updates:

```bash
DFLH_WEB_BASE_URL=http://127.0.0.1:4173 DFLH_WEB_VISUAL_MODE=capture node design-system/scripts/visual-check-web.mjs
```

`visual-check-web.mjs` provides deterministic API fixtures by default so screenshots capture populated News Feed, Messages, and My Page states instead of backend-error skeletons.
