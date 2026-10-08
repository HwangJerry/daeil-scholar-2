# Alumni filter multi-selection — 2026-10-08

The iOS and Android filter sheets follow [Figma](https://www.figma.com/design/Tex5GaxI9cSBCBu57oTccf/Untitled?node-id=3-2).
Deploy this backend before distributing the updated apps. No schema migration is required.

## Search contract

Authenticated `GET /api/alumni` accepts these optional comma-separated query values:

| Parameter | Values | Example |
| --- | --- | --- |
| `cohorts` | Cohort strings | `30,31` |
| `departments` | Canonical signup department strings | `영어,스페인어` |
| `jobCategories` | Positive category IDs | `2,3` |

HTTP clients must URL-encode query values. Repeated plural keys are flattened; repeated
legacy `cohort`, `department`, and `jobCategory` keys are also accepted. Existing single
keys remain compatible. A plural key takes precedence over its corresponding legacy key,
including an empty plural value. Blank strings, invalid job IDs and duplicates are removed.

Values within a group use OR (`IN`); groups and the name filter use AND. Empty groups
are unrestricted. Count and page queries use the same bound parameters and retain the
approved-verification, active-member and message-recipient restrictions. `page` starts at
1, default `size` is 20 and maximum is 50; native clients request 30. Response shape is unchanged.

## Native behavior

- Cohort: numeric input and Add, with normalized positive integer and duplicate validation.
- Departments: signup's 프랑스어, 독일어, 일본어, 중국어, 스페인어, 러시아어, 영어.
  Display names append 과. 전체 clears only department selections. Selecting individuals
  removes 전체; deselecting the last individual restores 전체.
- Occupation: select a pending category, then Add. Added options are disabled.
- Draft arrays and pending inputs remain local to the sheet. Apply commits all groups
  once and loads page 1. Close/back/swipe discards the draft. Pending values are not applied.
- Reset clears the draft and pending inputs while preserving the name query.
- Applied chips wrap. Removing a chip changes only that value and loads page 1.
- Body scrolls; footer stays above the native safe area and keyboard.

`/api/alumni/filters` continues to return directory options. Native department buttons
use the canonical signup list, so all seven remain available even when no matching members exist.

## Verification

Handler/service/repository regressions cover multi-selection, repeated legacy keys,
single-key compatibility, normalization, count/page consistency and 전체 departments.
Native unit and UI tests cover serialization, one-request Apply, numeric validation,
exclusive 전체, cancellation and individual chip removal.
