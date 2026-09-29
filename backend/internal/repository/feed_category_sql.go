// feed_category_sql.go — Shared SQL that resolves a WEO_BOARDBBS post (alias b) to its feed category
package repository

// feedCategoryJoin resolves b.FEED_CATEGORY_SEQ to its category (fc) and joins
// the single default category (fd). A NULL or dangling FEED_CATEGORY_SEQ falls
// back to fd, so every existing post counts as the default. fd is a one-row
// derived table (LEFT JOIN ON 1 = 1) so a post is never duplicated or dropped.
const feedCategoryJoin = `
	LEFT JOIN ALUMNI_FEED_CATEGORY fc ON fc.FC_SEQ = b.FEED_CATEGORY_SEQ
	LEFT JOIN (
		SELECT FC_SEQ, FC_CODE, FC_NAME FROM ALUMNI_FEED_CATEGORY
		WHERE IS_DEFAULT = 'Y' ORDER BY SORT_ORDER LIMIT 1
	) fd ON 1 = 1`

// feedCategoryColumns selects the resolved category code and name. Names are
// read at query time, so a rename shows everywhere at once. The literals only
// apply if the seeded default row were missing.
const feedCategoryColumns = `
	IFNULL(fc.FC_CODE, IFNULL(fd.FC_CODE, 'notice')) AS category,
	IFNULL(fc.FC_NAME, IFNULL(fd.FC_NAME, '공지')) AS category_name`

// feedCategorySeqExpr is the resolved category seq, for admin reads and the
// admin list filter.
const feedCategorySeqExpr = `IFNULL(fc.FC_SEQ, IFNULL(fd.FC_SEQ, 0))`
