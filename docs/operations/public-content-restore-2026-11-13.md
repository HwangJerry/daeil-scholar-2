# Production public-content restoration — 2026-11-13

Restored archived Hall of Fame and NOTICE content into `daeil_app_launch` following the user's narrowed scope. No legacy member, authentication, order, payment, comment or like records were imported.

| Data | Restored | Public |
| --- | ---: | ---: |
| Hall of Fame entries | 330 | 155 |
| Hall import audit batches | 1 | — |
| NOTICE posts, including newsletters | 128 | 96 |
| Referenced file metadata | 85 | Original visibility retained |
| Referenced physical assets | 88 | Original content references retained |

All Hall consent/visibility/deletion fields and all post publication flags were preserved. The remaining 32 posts remain private. Historical post authors map to the current administrator only where the legacy identity was explicitly confirmed; other historical member IDs become NULL while display bylines remain. Private post contact/password/IP metadata was omitted. Existing current profile-file rows were preserved exactly. Legacy inline-image (`B`) and attachment (`BB`) gates remain distinct. New attachment metadata IDs use `10000 + legacy ID` to avoid current profile-file ID collisions. Asset URLs now use the current `/files/` and `/uploads/` aliases.

## Evidence

- Source: `daeil_legacy_launch`; archive: `/app/legacy-archives/20261111T144150Z`.
- Private production maintenance directory: `/app/maintenance/public-content-restore-20261113-v2`.
- Plan SHA-256: `ff02d30c856b002b00a1afc4b3a5fd4aed7e373d344913bf39564b843ffad42f`.
- Rehearsed in a separate schema and compared every imported column before applying.
- Full pre-restore production database dump retained privately as `before.sql.gz`; gzip integrity passed.
- Inserts committed in one transaction; every restored row matched the saved plan immediately afterward.
- Post-API verification also passed, allowing only nondecreasing live view counters.
- All 88 assets (160,201,940 bytes) downloaded through production HTTPS and matched archive SHA-256 values.
- Public Hall API returned 155 entries; public hero returned October 2026 news with body content.
- Approved review account loaded the feed and newsletter 181 (March 2026); all 32 hidden posts denied access. Test session logged out successfully (HTTP response has no JSON body).
- Health endpoint returned 200. No application deployment or restart was needed.

The standalone `deploy/restore_public_content.py` supports plan, stage, rehearsal, apply and verification. It refuses nonempty target content, existing rehearsal databases, reused maintenance directories, plan digest mismatches and file overwrites. Plans, backups and raw member/contact data must remain outside Git. This is a migration for the observed source and target, not a general merge tool; do not rerun against nonempty production content.
