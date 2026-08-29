package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrActiveConflict = errors.New("active_conflict")

type AdminBannerAdRepository struct {
	DB *sqlx.DB
}

func NewAdminBannerAdRepository(db *sqlx.DB) *AdminBannerAdRepository {
	return &AdminBannerAdRepository{DB: db}
}

func (r *AdminBannerAdRepository) GetBanners() ([]model.AdminBannerAdRow, error) {
	banners := make([]model.AdminBannerAdRow, 0)
	if err := r.DB.Select(&banners, `
		SELECT BN_SEQ, BN_NAME, BN_URL, OPEN_YN, INDX,
		       DATE_FORMAT(BN_START_DATE, '%Y-%m-%dT%H:%i:%sZ') AS BN_START_DATE,
		       DATE_FORMAT(BN_END_DATE, '%Y-%m-%dT%H:%i:%sZ') AS BN_END_DATE,
		       DATE_FORMAT(CREATED_AT, '%Y-%m-%d %H:%i:%s') AS CREATED_AT,
		       DATE_FORMAT(UPDATED_AT, '%Y-%m-%d %H:%i:%s') AS UPDATED_AT
		FROM MAIN_BANNER_AD
		ORDER BY INDX ASC
	`); err != nil {
		return nil, err
	}
	if len(banners) == 0 {
		return banners, nil
	}

	bnSeqs := make([]int, 0, len(banners))
	indexBySeq := make(map[int]int, len(banners))
	for i := range banners {
		bnSeqs = append(bnSeqs, banners[i].BNSeq)
		banners[i].Images = make([]model.BannerAdImage, 0)
		indexBySeq[banners[i].BNSeq] = i
	}

	query, args, err := sqlx.In(`
		SELECT BNI_SEQ, BN_SEQ, IMAGE_URL, SORT_ORDER
		FROM MAIN_BANNER_AD_IMAGE
		WHERE BN_SEQ IN (?)
		ORDER BY BN_SEQ ASC, SORT_ORDER ASC
	`, bnSeqs)
	if err != nil {
		return nil, err
	}
	query = r.DB.Rebind(query)

	images := make([]model.BannerAdImage, 0)
	if err := r.DB.Select(&images, query, args...); err != nil {
		return nil, err
	}
	for _, image := range images {
		index, ok := indexBySeq[image.BNSeq]
		if !ok {
			continue
		}
		banners[index].Images = append(banners[index].Images, image)
	}

	statsQuery, statsArgs, err := sqlx.In(`
		SELECT BN_SEQ,
		       COALESCE(SUM(CASE WHEN LOG_TYPE = 'VIEW' THEN 1 ELSE 0 END), 0) AS VIEW_COUNT,
		       COALESCE(SUM(CASE WHEN LOG_TYPE = 'CLICK' THEN 1 ELSE 0 END), 0) AS CLICK_COUNT
		FROM WEO_BANNER_AD_LOG
		WHERE BN_SEQ IN (?)
		GROUP BY BN_SEQ
	`, bnSeqs)
	if err != nil {
		return nil, err
	}
	statsQuery = r.DB.Rebind(statsQuery)

	stats := make([]model.AdminBannerAdStats, 0)
	if err := r.DB.Select(&stats, statsQuery, statsArgs...); err != nil {
		return nil, err
	}
	for _, bannerStats := range stats {
		index, ok := indexBySeq[bannerStats.BNSeq]
		if !ok {
			continue
		}
		banners[index].ViewCount = bannerStats.ViewCount
		banners[index].ClickCount = bannerStats.ClickCount
	}

	return banners, nil
}

func (r *AdminBannerAdRepository) GetBanner(seq int) (*model.AdminBannerAdRow, error) {
	var banner model.AdminBannerAdRow
	if err := r.DB.Get(&banner, `
		SELECT BN_SEQ, BN_NAME, BN_URL, OPEN_YN, INDX,
		       DATE_FORMAT(BN_START_DATE, '%Y-%m-%dT%H:%i:%sZ') AS BN_START_DATE,
		       DATE_FORMAT(BN_END_DATE, '%Y-%m-%dT%H:%i:%sZ') AS BN_END_DATE,
		       DATE_FORMAT(CREATED_AT, '%Y-%m-%d %H:%i:%s') AS CREATED_AT,
		       DATE_FORMAT(UPDATED_AT, '%Y-%m-%d %H:%i:%s') AS UPDATED_AT
		FROM MAIN_BANNER_AD
		WHERE BN_SEQ = ?
	`, seq); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	banner.Images = make([]model.BannerAdImage, 0)
	if err := r.DB.Select(&banner.Images, `
		SELECT BNI_SEQ, BN_SEQ, IMAGE_URL, SORT_ORDER
		FROM MAIN_BANNER_AD_IMAGE
		WHERE BN_SEQ = ?
		ORDER BY SORT_ORDER ASC
	`, seq); err != nil {
		return nil, err
	}
	stats, err := r.GetBannerStats(seq)
	if err != nil {
		return nil, err
	}
	banner.ViewCount = stats.ViewCount
	banner.ClickCount = stats.ClickCount
	return &banner, nil
}

func (r *AdminBannerAdRepository) InsertBanner(a *model.AdminBannerAdInsert) (int, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if a.OpenYN == "Y" {
		count, err := lockOverlappingActiveBanners(tx, 0, a.BNStartDate, a.BNEndDate)
		if err != nil {
			return 0, err
		}
		if count > 0 {
			return 0, ErrActiveConflict
		}
	}

	res, err := tx.Exec(`
		INSERT INTO MAIN_BANNER_AD (
			BN_NAME, BN_URL, OPEN_YN, INDX, BN_START_DATE, BN_END_DATE, CREATED_AT, UPDATED_AT
		) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
	`, a.BNName, a.BNURL, a.OpenYN, a.Indx, a.BNStartDate, a.BNEndDate)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for i, imageURL := range a.ImageURLs {
		if _, err := tx.Exec(`
			INSERT INTO MAIN_BANNER_AD_IMAGE (BN_SEQ, IMAGE_URL, SORT_ORDER)
			VALUES (?, ?, ?)
		`, id, imageURL, i); err != nil {
			return 0, err
		}
		if err := attachBannerUpload(tx, int(id), imageURL); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *AdminBannerAdRepository) UpdateBanner(seq int, a *model.AdminBannerAdInsert) ([]string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	previousImageURLs := make([]string, 0)
	if err := tx.Select(&previousImageURLs, `
		SELECT IMAGE_URL
		FROM MAIN_BANNER_AD_IMAGE
		WHERE BN_SEQ = ?
		FOR UPDATE
	`, seq); err != nil {
		return nil, err
	}

	if a.OpenYN == "Y" {
		count, err := lockOverlappingActiveBanners(tx, seq, a.BNStartDate, a.BNEndDate)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrActiveConflict
		}
	}

	if _, err := tx.Exec(`
		UPDATE MAIN_BANNER_AD
		SET BN_NAME = ?, BN_URL = ?, OPEN_YN = ?, INDX = ?, BN_START_DATE = ?, BN_END_DATE = ?, UPDATED_AT = NOW()
		WHERE BN_SEQ = ?
	`, a.BNName, a.BNURL, a.OpenYN, a.Indx, a.BNStartDate, a.BNEndDate, seq); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(`DELETE FROM MAIN_BANNER_AD_IMAGE WHERE BN_SEQ = ?`, seq); err != nil {
		return nil, err
	}

	for i, imageURL := range a.ImageURLs {
		if _, err := tx.Exec(`
			INSERT INTO MAIN_BANNER_AD_IMAGE (BN_SEQ, IMAGE_URL, SORT_ORDER)
			VALUES (?, ?, ?)
		`, seq, imageURL, i); err != nil {
			return nil, err
		}
		if err := attachBannerUpload(tx, seq, imageURL); err != nil {
			return nil, err
		}
	}

	removedImageURLs, err := deleteUnreferencedBannerUploads(tx, removedURLs(previousImageURLs, a.ImageURLs))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return removedImageURLs, nil
}

func (r *AdminBannerAdRepository) DeleteBanner(seq int) ([]string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	imageURLs := make([]string, 0)
	if err := tx.Select(&imageURLs, `
		SELECT IMAGE_URL
		FROM MAIN_BANNER_AD_IMAGE
		WHERE BN_SEQ = ?
		FOR UPDATE
	`, seq); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM MAIN_BANNER_AD WHERE BN_SEQ = ?`, seq); err != nil {
		return nil, err
	}

	removedImageURLs, err := deleteUnreferencedBannerUploads(tx, imageURLs)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return removedImageURLs, nil
}

func attachBannerUpload(tx *sqlx.Tx, bnSeq int, imageURL string) error {
	_, err := tx.Exec(`
		UPDATE WEO_FILES
		SET F_JOIN_SEQ = ?
		WHERE F_GATE = 'BA'
		  AND CONCAT(TRIM(TRAILING '/' FROM IFNULL(FILE_PATH, '')), '/', FILE_NAME) = ?
	`, bnSeq, imageURL)
	return err
}

func removedURLs(previousURLs, currentURLs []string) []string {
	current := make(map[string]struct{}, len(currentURLs))
	for _, imageURL := range currentURLs {
		current[imageURL] = struct{}{}
	}

	removed := make([]string, 0)
	seen := make(map[string]struct{}, len(previousURLs))
	for _, imageURL := range previousURLs {
		if _, kept := current[imageURL]; kept {
			continue
		}
		if _, duplicate := seen[imageURL]; duplicate {
			continue
		}
		seen[imageURL] = struct{}{}
		removed = append(removed, imageURL)
	}
	return removed
}

func deleteUnreferencedBannerUploads(tx *sqlx.Tx, candidateURLs []string) ([]string, error) {
	unreferenced := make([]string, 0, len(candidateURLs))
	seen := make(map[string]struct{}, len(candidateURLs))
	for _, imageURL := range candidateURLs {
		if _, duplicate := seen[imageURL]; duplicate {
			continue
		}
		seen[imageURL] = struct{}{}

		var referenceCount int
		if err := tx.Get(&referenceCount, `
			SELECT COUNT(*)
			FROM MAIN_BANNER_AD_IMAGE
			WHERE IMAGE_URL = ?
		`, imageURL); err != nil {
			return nil, err
		}
		if referenceCount > 0 {
			continue
		}
		if _, err := tx.Exec(`
			DELETE FROM WEO_FILES
			WHERE F_GATE = 'BA'
			  AND CONCAT(TRIM(TRAILING '/' FROM IFNULL(FILE_PATH, '')), '/', FILE_NAME) = ?
		`, imageURL); err != nil {
			return nil, err
		}
		unreferenced = append(unreferenced, imageURL)
	}
	return unreferenced, nil
}

func lockOverlappingActiveBanners(tx *sqlx.Tx, excludeSeq int, startDate, endDate *string) (int, error) {
	var count int
	if err := tx.Get(&count, `
		SELECT COUNT(*)
		FROM MAIN_BANNER_AD
		WHERE OPEN_YN = 'Y'
		  AND BN_SEQ != ?
		  AND (BN_START_DATE IS NULL OR ? IS NULL OR BN_START_DATE <= ?)
		  AND (BN_END_DATE IS NULL OR ? IS NULL OR BN_END_DATE >= ?)
		FOR UPDATE
	`, excludeSeq, endDate, endDate, startDate, startDate); err != nil {
		return 0, err
	}
	return count, nil
}

func bannerAdPeriodsOverlap(existingStart, existingEnd, candidateStart, candidateEnd *string) bool {
	startsBeforeCandidateEnds := existingStart == nil || candidateEnd == nil || *existingStart <= *candidateEnd
	endsAfterCandidateStarts := existingEnd == nil || candidateStart == nil || *existingEnd >= *candidateStart
	return startsBeforeCandidateEnds && endsAfterCandidateStarts
}

func (r *AdminBannerAdRepository) GetBannerStats(bnSeq int) (*model.AdminBannerAdStats, error) {
	stats := &model.AdminBannerAdStats{}
	if err := r.DB.Get(stats, `
		SELECT ? AS BN_SEQ,
		       COALESCE(SUM(CASE WHEN LOG_TYPE = 'VIEW' THEN 1 ELSE 0 END), 0) AS VIEW_COUNT,
		       COALESCE(SUM(CASE WHEN LOG_TYPE = 'CLICK' THEN 1 ELSE 0 END), 0) AS CLICK_COUNT
		FROM WEO_BANNER_AD_LOG
		WHERE BN_SEQ = ?
	`, bnSeq, bnSeq); err != nil {
		return nil, err
	}
	return stats, nil
}
