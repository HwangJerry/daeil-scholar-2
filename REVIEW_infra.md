# Infrastructure Review — reviewer-infra
Date: 2026-02-20

---

## Task #4 — nginx /pg/ rate limit removal (commit 04ac789)

**Verdict: PASS**

Checks against `nginx/conf.d/alumni.conf`:

1. **`/pg/` has no `limit_req`** — Confirmed. Lines 49–55 show only `proxy_pass` and `proxy_set_header` directives. No rate limiting present.

2. **Other `limit_req` blocks intact** — Both previously rate-limited locations still have their directives:
   - `/api/` (line 34): `limit_req zone=api burst=50 nodelay;` ✓
   - `/api/auth/` (line 43): `limit_req zone=login burst=3 nodelay;` ✓

3. **Syntax valid** — All server blocks open and close with matching braces. All directives end with semicolons. No syntax issues found.

4. **No unrelated changes** — The rest of the file (`/`, `/admin/`, `/legacy/`, `/files/`, `/uploads/` blocks) is untouched.

---

## Task #5 — deploy.sh migration docs (commit 35920ea)

**Verdict: PASS**

Checks against `deploy.sh`:

1. **Comment block present** — Lines 9–30 contain a clearly demarcated `DATABASE MIGRATIONS — MANUAL STEP REQUIRED` comment block. ✓

2. **Migrations 001–010 all listed** — Each migration file (001 through 010) is explicitly named within the block (lines 18–28). ✓

3. **mysql command syntax included** — Line 15 provides the command template:
   `mysql -u USER -p DB_NAME < backend/migrations/NNN_name.sql` ✓

4. **Sequential requirement stated** — Line 29 reads: "Migrations MUST be applied sequentially (in numbered order)." ✓

5. **Deploy logic unchanged** — Lines 32–65 (Go build, SPA builds, scp uploads, systemctl restart, nginx reload) are identical to the prior version. No functional changes. ✓

---

## Overall Verdict: APPROVED

Both commits are clean, scoped, and correct. No issues found.
