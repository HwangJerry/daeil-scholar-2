// visit_service.go — Business logic for visit tracking; in-memory dedupe, hashing, DAU/MAU queries
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// kstZone is the fixed Asia/Seoul offset used for all "today" boundaries so
// date rollovers are deterministic regardless of the host machine's TZ setting.
var kstZone = time.FixedZone("KST", 9*3600)

// visitCacheTTL is how long a (date, visitor) tuple is treated as already-recorded
// in process memory before the next beacon triggers another UPSERT.
const visitCacheTTL = 12 * time.Hour

type VisitService struct {
	repo   *repository.VisitRepository
	cache  *cache.Cache
	salt   string
	logger zerolog.Logger
}

// NewVisitService wires the repo, shared cache, and IP hashing salt. If the
// provided salt is empty, a random per-process salt is generated; visitor IP
// hashes will then differ across restarts, which is acceptable because they
// are only used for basic duplicate detection, not cross-session identity.
func NewVisitService(repo *repository.VisitRepository, cacheStore *cache.Cache, salt string, logger zerolog.Logger) *VisitService {
	effectiveSalt := salt
	if effectiveSalt == "" {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err == nil {
			effectiveSalt = hex.EncodeToString(buf)
		}
		logger.Warn().Msg("VISIT_IP_SALT not set; using random per-process salt")
	}
	return &VisitService{
		repo:   repo,
		cache:  cacheStore,
		salt:   effectiveSalt,
		logger: logger,
	}
}

// RecordVisit records a beacon hit, deduplicating repeat hits from the same
// visitor within visitCacheTTL so we don't hammer the DB on every page view.
func (s *VisitService) RecordVisit(visitorID string, usrSeq int, userAgent, ipAddr string) error {
	if visitorID == "" {
		return fmt.Errorf("visit: empty visitorID")
	}
	now := time.Now().In(kstZone)
	dateKey := now.Format("2006-01-02")
	cacheKey := fmt.Sprintf("visit:%s:%d:%s", dateKey, usrSeq, visitorID)
	if _, found := s.cache.Get(cacheKey); found {
		return nil
	}
	s.cache.Set(cacheKey, 1, visitCacheTTL)

	return s.repo.UpsertDaily(now, visitorID, usrSeq, hashTrunc(userAgent), hashTrunc(ipAddr+s.salt))
}

// ActiveUsers assembles the admin chart response: a series of (date, DAU, MAU)
// points for the requested range, plus today's DAU and the current rolling MAU.
func (s *VisitService) ActiveUsers(from, to time.Time) (*model.ActiveUsersResponse, error) {
	rows, err := s.repo.GetSummaryRange(from, to)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]model.VisitSummary, len(rows))
	for _, row := range rows {
		byDate[row.Date.Format("2006-01-02")] = row
	}

	points := make([]model.ActiveUsersPoint, 0)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if row, ok := byDate[key]; ok {
			points = append(points, model.ActiveUsersPoint{Date: key, DAU: row.DAUTotal, MAU: row.MAUTotal})
		} else {
			points = append(points, model.ActiveUsersPoint{Date: key, DAU: 0, MAU: 0})
		}
	}

	today := Today()
	dauToday, _, _, err := s.repo.CountDAU(today)
	if err != nil {
		return nil, err
	}
	mauCurrent, err := s.repo.CountMAU(today)
	if err != nil {
		return nil, err
	}

	return &model.ActiveUsersResponse{
		Points:     points,
		DAUToday:   dauToday,
		MAUCurrent: mauCurrent,
	}, nil
}

// DashboardCounts returns just today's DAU and the current rolling MAU, used by
// the main admin dashboard summary endpoint.
func (s *VisitService) DashboardCounts() (dau, mau uint32, err error) {
	today := Today()
	dau, _, _, err = s.repo.CountDAU(today)
	if err != nil {
		return 0, 0, err
	}
	mau, err = s.repo.CountMAU(today)
	return
}

// Today returns the current KST-aligned calendar date at midnight. Shared so
// the aggregation job and handlers agree on what "today" means.
func Today() time.Time {
	now := time.Now().In(kstZone)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, kstZone)
}

func hashTrunc(v string) string {
	if v == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])[:16]
}
