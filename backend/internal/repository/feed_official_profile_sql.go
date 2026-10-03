// feed_official_profile_sql.go — Shared SQL that exposes a WEO_BOARDBBS post's (alias b) official profile flag
package repository

// officialProfileColumn selects whether a post was published under the
// foundation's official profile (migration 080). The comparison yields 1/0, so
// it scans straight into a bool; every pre-080 post is 'N' and reads false.
const officialProfileColumn = `(b.OFFICIAL_PROFILE_YN = 'Y') AS official_profile`
