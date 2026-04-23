package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"back/internal/ent"
)

var ErrInvalidDashboardRange = errors.New("invalid dashboard range")

type cachedResult struct {
	data      *DashboardStatsResponse
	expiresAt time.Time
}

type DashboardService struct {
	client *ent.Client
	db     *sql.DB
	cache  sync.Map // key: cacheKey, value: cachedResult
}

func NewDashboardService(client *ent.Client, db *sql.DB) *DashboardService {
	return &DashboardService{
		client: client,
		db:     db,
	}
}

type DashboardFilters struct {
	BranchID  *int
	Range     string
	StartDate *time.Time
	EndDate   *time.Time
}

type DashboardStatsResponse struct {
	Filters DashboardFiltersResponse `json:"filters"`
	Summary DashboardSummary         `json:"summary"`
	Charts  DashboardCharts          `json:"charts"`
}

type DashboardFiltersResponse struct {
	BranchID  *int       `json:"branch_id,omitempty"`
	Range     string     `json:"range"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type DashboardSummary struct {
	MarkingsToday          int `json:"markings_today"`
	PeopleInside           int `json:"people_inside"`
	LateArrivals           int `json:"late_arrivals"`
	Alerts                 int `json:"alerts"`
	JustifiedAbsences      int `json:"justified_absences"`
	MarkingsVsYesterdayPct int `json:"markings_vs_yesterday_pct"`
}

type DashboardCharts struct {
	MarkingsLast7Days []ChartPoint            `json:"markings_last_7_days"`
	EntriesVsExits    EntriesVsExits          `json:"entries_vs_exits"`
	CurrentStatus     CurrentStatus           `json:"current_status"`
	ActivityByHour    []ChartPoint            `json:"activity_by_hour"`
	TopLates          []TopLateItem           `json:"top_lates"`
	TopBranches       []TopBranchMovementItem `json:"top_branches"`
}

type DashboardLiveFilters struct {
	BranchID  *int       `json:"branch_id,omitempty"`
	Limit     int        `json:"limit"`
	Range     string     `json:"range"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type DashboardLiveResponse struct {
	Filters   DashboardLiveFiltersResponse `json:"filters"`
	LastMarks []DashboardLastMarkItem      `json:"last_marks"`
	InsideNow []DashboardInsideNowItem     `json:"inside_now"`
}

type DashboardLiveFiltersResponse struct {
	BranchID  *int       `json:"branch_id,omitempty"`
	Limit     int        `json:"limit"`
	Range     string     `json:"range"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type DashboardLastMarkItem struct {
	MarkingID  int       `json:"marking_id"`
	UserID     int       `json:"user_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	MarkedAt   time.Time `json:"marked_at"`
	BranchID   int       `json:"branch_id"`
	BranchName string    `json:"branch_name"`
}

type DashboardInsideNowItem struct {
	UserID     int       `json:"user_id"`
	Name       string    `json:"name"`
	EnteredAt  time.Time `json:"entered_at"`
	BranchID   int       `json:"branch_id"`
	BranchName string    `json:"branch_name"`
}

type ChartPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type EntriesVsExits struct {
	Entries int `json:"entries"`
	Exits   int `json:"exits"`
}

type CurrentStatus struct {
	Inside  int `json:"inside"`
	Outside int `json:"outside"`
	NoMark  int `json:"no_mark"`
}

type TopLateItem struct {
	UserID      int    `json:"user_id"`
	Name        string `json:"name"`
	Branch      string `json:"branch"`
	MinutesLate int    `json:"minutes_late"`
}

type TopBranchMovementItem struct {
	BranchID int    `json:"branch_id"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
}

// Punctuality

type TodayPunctualityItem struct {
	UserID           int        `json:"user_id"`
	Name             string     `json:"name"`
	BranchID         int        `json:"branch_id"`
	BranchName       string     `json:"branch_name"`
	ShiftName        string     `json:"shift_name"`
	StartTime        string     `json:"start_time"`
	EndTime          string     `json:"end_time"`
	BreakMinutes     int        `json:"break_minutes"`
	WorkInAt         *time.Time `json:"work_in_at"`
	BreakOutAt       *time.Time `json:"break_out_at"`
	BreakInAt        *time.Time `json:"break_in_at"`
	WorkOutAt        *time.Time `json:"work_out_at"`
	EntryDiffMinutes *int       `json:"entry_diff_minutes"` // positive = late, negative = early
	BreakDiffMinutes *int       `json:"break_diff_minutes"` // positive = over break, negative = early
	ExitDiffMinutes  *int       `json:"exit_diff_minutes"`  // positive = overtime, negative = left early
	EntryStatus      string     `json:"entry_status"`       // "on_time"|"late"|"early"|"no_mark"
	BreakStatus      string     `json:"break_status"`       // "on_time"|"over"|"early"|"no_break"
	ExitStatus       string     `json:"exit_status"`        // "on_time"|"overtime"|"early"|"no_mark"
}

type DashboardPunctualityResponse struct {
	BranchID *int                   `json:"branch_id,omitempty"`
	Date     string                 `json:"date"`
	Items    []TodayPunctualityItem `json:"items"`
	Total    int                    `json:"total"`
}

func (s *DashboardService) GetStats(ctx context.Context, f DashboardFilters) (*DashboardStatsResponse, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("stats_%v_%s_%v_%v", f.BranchID, f.Range, f.StartDate, f.EndDate)

	// Check cache
	if cached, ok := s.cache.Load(cacheKey); ok {
		if result := cached.(cachedResult); time.Now().Before(result.expiresAt) {
			return result.data, nil
		}
		// Remove expired entry
		s.cache.Delete(cacheKey)
	}

	// Compute result
	resp, err := s.computeStats(ctx, f)
	if err != nil {
		return nil, err
	}

	// Cache result for 5 minutes
	s.cache.Store(cacheKey, cachedResult{
		data:      resp,
		expiresAt: time.Now().Add(5 * time.Minute),
	})

	return resp, nil
}

func (s *DashboardService) GetLiveData(ctx context.Context, f DashboardLiveFilters) (*DashboardLiveResponse, error) {
	if f.Limit <= 0 {
		f.Limit = 5
	}

	var start, end time.Time
	var err error
	if f.StartDate != nil && f.EndDate != nil {
		start = *f.StartDate
		end = f.EndDate.Add(24 * time.Hour)
	} else {
		if f.Range == "" {
			f.Range = "today"
		}
		start, end, err = resolveRange(f.Range)
		if err != nil {
			return nil, err
		}
	}

	lastMarks, err := s.getLastMarks(ctx, f.BranchID, start, end, f.Limit)
	if err != nil {
		return nil, err
	}

	insideNow, err := s.getInsideNow(ctx, f.BranchID, start, end, f.Limit)
	if err != nil {
		return nil, err
	}

	return &DashboardLiveResponse{
		Filters: DashboardLiveFiltersResponse{
			BranchID:  f.BranchID,
			Limit:     f.Limit,
			Range:     f.Range,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
		},
		LastMarks: lastMarks,
		InsideNow: insideNow,
	}, nil
}

func (s *DashboardService) computeStats(ctx context.Context, f DashboardFilters) (*DashboardStatsResponse, error) {

	var start, end time.Time
	var err error

	if f.StartDate != nil && f.EndDate != nil {
		start = *f.StartDate
		end = f.EndDate.Add(24 * time.Hour) // include the end date
	} else {
		start, end, err = resolveRange(f.Range)
		if err != nil {
			return nil, err
		}
	}

	todayStart, todayEnd := dayBounds(time.Now())
	yesterdayStart, yesterdayEnd := dayBounds(time.Now().AddDate(0, 0, -1))

	summary, err := s.getSummary(ctx, f.BranchID, todayStart, todayEnd, yesterdayStart, yesterdayEnd)
	if err != nil {
		return nil, err
	}

	markings7d, err := s.getMarkingsByRange(ctx, f.BranchID, start, end)
	if err != nil {
		return nil, err
	}

	// entries/exits and activity are calculated over the filtered range
	entriesVsExits, err := s.getEntriesVsExits(ctx, f.BranchID, start, end)
	if err != nil {
		return nil, err
	}

	// current_status is always real-time (today)
	currentStatus, err := s.getCurrentStatus(ctx, f.BranchID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}

	activityByHour, err := s.getActivityByHour(ctx, f.BranchID, start, end)
	if err != nil {
		return nil, err
	}

	topLates, err := s.getTopLates(ctx, f.BranchID, start, end)
	if err != nil {
		return nil, err
	}

	topBranches, err := s.getTopBranches(ctx, f.BranchID, start, end)
	if err != nil {
		return nil, err
	}

	if topLates == nil {
		topLates = []TopLateItem{}
	}
	if topBranches == nil {
		topBranches = []TopBranchMovementItem{}
	}
	if markings7d == nil {
		markings7d = []ChartPoint{}
	}
	if activityByHour == nil {
		activityByHour = []ChartPoint{}
	}

	return &DashboardStatsResponse{
		Filters: DashboardFiltersResponse{
			BranchID:  f.BranchID,
			Range:     f.Range,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
		},
		Summary: summary,
		Charts: DashboardCharts{
			MarkingsLast7Days: markings7d,
			EntriesVsExits:    entriesVsExits,
			CurrentStatus:     currentStatus,
			ActivityByHour:    activityByHour,
			TopLates:          topLates,
			TopBranches:       topBranches,
		},
	}, nil
}

func resolveRange(r string) (time.Time, time.Time, error) {
	now := time.Now()
	loc := now.Location()

	switch r {
	case "", "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end := start.Add(24 * time.Hour)
		return start, end, nil
	case "last_7_days":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
		start := end.AddDate(0, 0, -7)
		return start, end, nil
	case "last_30_days":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
		start := end.AddDate(0, 0, -30)
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, ErrInvalidDashboardRange
	}
}

func dayBounds(t time.Time) (time.Time, time.Time) {
	loc := t.Location()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start, start.Add(24 * time.Hour)
}

func (s *DashboardService) getSummary(
	ctx context.Context,
	branchID *int,
	todayStart, todayEnd, yesterdayStart, yesterdayEnd time.Time,
) (DashboardSummary, error) {
	markingsToday, err := s.countMarkings(ctx, branchID, todayStart, todayEnd)
	if err != nil {
		return DashboardSummary{}, err
	}

	markingsYesterday, err := s.countMarkings(ctx, branchID, yesterdayStart, yesterdayEnd)
	if err != nil {
		return DashboardSummary{}, err
	}

	peopleInside, err := s.countPeopleInside(ctx, branchID, todayStart, todayEnd)
	if err != nil {
		return DashboardSummary{}, err
	}

	lateArrivals, err := s.countLateArrivals(ctx, branchID, todayStart, todayEnd)
	if err != nil {
		return DashboardSummary{}, err
	}

	alerts, err := s.countAlerts(ctx, branchID, todayStart, todayEnd)
	if err != nil {
		return DashboardSummary{}, err
	}

	justifiedAbsences, err := s.countJustifiedAbsences(ctx, branchID, todayStart, todayEnd)
	if err != nil {
		return DashboardSummary{}, err
	}

	pct := 0
	if markingsYesterday > 0 {
		pct = int((float64(markingsToday-markingsYesterday) / float64(markingsYesterday)) * 100.0)
		if pct < -100 {
			pct = -100
		}
	}

	return DashboardSummary{
		MarkingsToday:          markingsToday,
		PeopleInside:           peopleInside,
		LateArrivals:           lateArrivals,
		Alerts:                 alerts,
		JustifiedAbsences:      justifiedAbsences,
		MarkingsVsYesterdayPct: pct,
	}, nil
}

func (s *DashboardService) countMarkings(ctx context.Context, branchID *int, start, end time.Time) (int, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT COUNT(*) AS total
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND (ad.work_in_at IS NOT NULL OR ad.work_out_at IS NOT NULL)
		%s
	`, branchWhere)

	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *DashboardService) countPeopleInside(ctx context.Context, branchID *int, start, end time.Time) (int, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND ad.work_in_at IS NOT NULL
		  AND ad.work_out_at IS NULL
		  %s
	`, branchWhere)

	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *DashboardService) countLateArrivals(ctx context.Context, branchID *int, start, end time.Time) (int, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ub.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM attendance_days ad
		JOIN user_branches ub ON ub.user_id = ad.user_id AND ub.is_active = true
		JOIN user_shift_assignments usa ON usa.user_id = ad.user_id AND usa.is_active = true
		JOIN shifts sh ON sh.id = usa.shift_id AND sh.is_active = true
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND ad.work_in_at IS NOT NULL
		  AND ad.work_in_at::time > sh.start_time::time
		  %s
	`, branchWhere)

	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *DashboardService) countAlerts(ctx context.Context, branchID *int, start, end time.Time) (int, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND (
			(ad.work_in_at IS NOT NULL AND ad.work_out_at IS NULL)
			OR
			(ad.work_in_at IS NULL AND ad.work_out_at IS NOT NULL)
		  )
		  %s
	`, branchWhere)

	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *DashboardService) countJustifiedAbsences(ctx context.Context, branchID *int, start, end time.Time) (int, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND ad.work_in_at IS NULL
		  %s
	`, branchWhere)

	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *DashboardService) getMarkingsByRange(ctx context.Context, branchID *int, start, end time.Time) ([]ChartPoint, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT DATE(ad.work_date) AS d,
		       COUNT(*) AS total
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND (ad.work_in_at IS NOT NULL OR ad.work_out_at IS NOT NULL)
		%s
		GROUP BY DATE(ad.work_date)
		ORDER BY DATE(ad.work_date)
	`, branchWhere)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := map[string]int{}
	for rows.Next() {
		var d time.Time
		var total int
		if err := rows.Scan(&d, &total); err != nil {
			return nil, err
		}
		found[d.Format("2006-01-02")] = total
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels := map[time.Weekday]string{
		time.Monday:    "Lun",
		time.Tuesday:   "Mar",
		time.Wednesday: "Mié",
		time.Thursday:  "Jue",
		time.Friday:    "Vie",
		time.Saturday:  "Sáb",
		time.Sunday:    "Dom",
	}

	days := int(end.Sub(start).Hours() / 24)
	if days < 1 {
		days = 1
	}

	points := make([]ChartPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		points = append(points, ChartPoint{
			Label: labels[d.Weekday()],
			Value: found[d.Format("2006-01-02")],
		})
	}

	return points, nil
}

func (s *DashboardService) getEntriesVsExits(ctx context.Context, branchID *int, start, end time.Time) (EntriesVsExits, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN ad.work_in_at IS NOT NULL THEN 1 ELSE 0 END), 0) AS entries_count,
			COALESCE(SUM(CASE WHEN ad.work_out_at IS NOT NULL THEN 1 ELSE 0 END), 0) AS exits_count
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		%s
	`, branchWhere)

	var res EntriesVsExits
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&res.Entries, &res.Exits)
	return res, err
}

func (s *DashboardService) getCurrentStatus(ctx context.Context, branchID *int, start, end time.Time) (CurrentStatus, error) {
	inside, err := s.countPeopleInside(ctx, branchID, start, end)
	if err != nil {
		return CurrentStatus{}, err
	}

	outsideArgs := []any{start, end}
	outsideBranchWhere := buildBranchFilter("ub.branch_id", branchID, &outsideArgs)

	outsideQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM attendance_days ad
		JOIN user_branches ub ON ub.user_id = ad.user_id AND ub.is_active = true
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND ad.work_in_at IS NOT NULL
		  AND ad.work_out_at IS NOT NULL
		  %s
	`, outsideBranchWhere)

	var outside int
	if err := s.db.QueryRowContext(ctx, outsideQuery, outsideArgs...).Scan(&outside); err != nil {
		return CurrentStatus{}, err
	}

	noMarkArgs := []any{}
	noMarkBranchWhere := buildBranchFilter("ub.branch_id", branchID, &noMarkArgs)
	noMarkArgs = append(noMarkArgs, start, end)

	noMarkQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM users u
		JOIN user_branches ub ON ub.user_id = u.id AND ub.is_active = true
		WHERE u.is_active = true
		  %s
		  AND NOT EXISTS (
			  SELECT 1
			  FROM attendance_days ad
			  WHERE ad.user_id = u.id
				AND ad.work_date >= $%d AND ad.work_date < $%d
		  )
	`, noMarkBranchWhere, len(noMarkArgs)-1, len(noMarkArgs))

	var noMark int
	if err := s.db.QueryRowContext(ctx, noMarkQuery, noMarkArgs...).Scan(&noMark); err != nil {
		return CurrentStatus{}, err
	}

	return CurrentStatus{
		Inside:  inside,
		Outside: outside,
		NoMark:  noMark,
	}, nil
}

func (s *DashboardService) getActivityByHour(ctx context.Context, branchID *int, start, end time.Time) ([]ChartPoint, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT EXTRACT(HOUR FROM ad.work_in_at)::int AS h,
		       COUNT(*) AS total
		FROM attendance_days ad
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND ad.work_in_at IS NOT NULL
		  %s
		GROUP BY EXTRACT(HOUR FROM ad.work_in_at)
		ORDER BY EXTRACT(HOUR FROM ad.work_in_at)
	`, branchWhere)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := map[int]int{}
	for rows.Next() {
		var hour int
		var total int
		if err := rows.Scan(&hour, &total); err != nil {
			return nil, err
		}
		found[hour] = total
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	points := make([]ChartPoint, 0, 24)
	for h := 0; h < 24; h++ {
		points = append(points, ChartPoint{
			Label: twoDigits(h),
			Value: found[h],
		})
	}
	return points, nil
}

func (s *DashboardService) getTopLates(ctx context.Context, branchID *int, start, end time.Time) ([]TopLateItem, error) {
	args := []any{start, end}
	branchWhere := buildBranchFilter("ub.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT
			u.id,
			TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS full_name,
			b.name AS branch_name,
			(EXTRACT(EPOCH FROM (ad.work_in_at - (ad.work_date::timestamp + sh.start_time::time))) / 60)::int AS minutes_late
		FROM attendance_days ad
		JOIN users u ON u.id = ad.user_id
		JOIN user_branches ub ON ub.user_id = u.id AND ub.is_active = true
		JOIN branches b ON b.id = ub.branch_id
		JOIN user_shift_assignments usa ON usa.user_id = u.id AND usa.is_active = true
		JOIN shifts sh ON sh.id = usa.shift_id AND sh.is_active = true
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND ad.work_in_at IS NOT NULL
		  AND ad.work_in_at > (ad.work_date::timestamp + sh.start_time::time)
		  %s
		ORDER BY minutes_late DESC
		LIMIT 5
	`, branchWhere)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TopLateItem, 0)
	for rows.Next() {
		var it TopLateItem
		if err := rows.Scan(&it.UserID, &it.Name, &it.Branch, &it.MinutesLate); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *DashboardService) getTopBranches(ctx context.Context, branchID *int, start, end time.Time) ([]TopBranchMovementItem, error) {
	query := `
		SELECT
			b.id,
			b.name,
			COALESCE(SUM(
				(CASE WHEN ad.work_in_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.work_out_at IS NOT NULL THEN 1 ELSE 0 END)
			), 0) AS total
		FROM branches b
		LEFT JOIN access_points ap ON ap.branch_id = b.id
		LEFT JOIN attendance_days ad ON ad.access_point_id = ap.id
			AND ad.work_date >= $1 AND ad.work_date < $2
		WHERE $3::int IS NULL OR b.id = $3
		GROUP BY b.id, b.name
		ORDER BY total DESC, b.name ASC
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, query, start, end, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TopBranchMovementItem, 0)
	for rows.Next() {
		var it TopBranchMovementItem
		if err := rows.Scan(&it.BranchID, &it.Name, &it.Count); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *DashboardService) getLastMarks(ctx context.Context, branchID *int, start, end time.Time, limit int) ([]DashboardLastMarkItem, error) {
	args := []any{limit, start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT marking_id, user_id, name, type, marked_at, branch_id, branch_name
		FROM (
			SELECT (ad.id * 10 + 1) AS marking_id,
				u.id AS user_id,
				CONCAT_WS(' ', u.first_name, u.last_name) AS name,
				'entry' AS type,
				ad.work_in_at AS marked_at,
				ad.branch_id,
				b.name AS branch_name
			FROM attendance_days ad
			JOIN users u ON u.id = ad.user_id
			JOIN branches b ON b.id = ad.branch_id
			WHERE ad.work_in_at IS NOT NULL
			  AND ad.work_date >= $2 AND ad.work_date < $3
			%s
			UNION ALL
			SELECT (ad.id * 10 + 2) AS marking_id,
				u.id AS user_id,
				CONCAT_WS(' ', u.first_name, u.last_name) AS name,
				'exit' AS type,
				ad.work_out_at AS marked_at,
				ad.branch_id,
				b.name AS branch_name
			FROM attendance_days ad
			JOIN users u ON u.id = ad.user_id
			JOIN branches b ON b.id = ad.branch_id
			WHERE ad.work_out_at IS NOT NULL
			  AND ad.work_date >= $2 AND ad.work_date < $3
			%s
		) AS marks
		ORDER BY marked_at DESC
		LIMIT $1
	`, branchWhere, branchWhere)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]DashboardLastMarkItem, 0)
	for rows.Next() {
		var it DashboardLastMarkItem
		if err := rows.Scan(&it.MarkingID, &it.UserID, &it.Name, &it.Type, &it.MarkedAt, &it.BranchID, &it.BranchName); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *DashboardService) getInsideNow(ctx context.Context, branchID *int, start, end time.Time, limit int) ([]DashboardInsideNowItem, error) {
	args := []any{limit, start, end}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		SELECT u.id AS user_id,
			CONCAT_WS(' ', u.first_name, u.last_name) AS name,
			ad.work_in_at AS entered_at,
			ad.branch_id,
			b.name AS branch_name
		FROM attendance_days ad
		JOIN users u ON u.id = ad.user_id
		JOIN branches b ON b.id = ad.branch_id
		WHERE ad.work_in_at IS NOT NULL
		  AND ad.work_out_at IS NULL
		  AND ad.work_date >= $2 AND ad.work_date < $3
		%s
		ORDER BY ad.work_in_at DESC
		LIMIT $1
	`, branchWhere)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]DashboardInsideNowItem, 0)
	for rows.Next() {
		var it DashboardInsideNowItem
		if err := rows.Scan(&it.UserID, &it.Name, &it.EnteredAt, &it.BranchID, &it.BranchName); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func buildBranchFilter(column string, branchID *int, args *[]any) string {
	if branchID == nil {
		return ""
	}
	*args = append(*args, *branchID)
	return fmt.Sprintf(" AND %s = $%d", column, len(*args))
}

func twoDigits(n int) string {
	return fmt.Sprintf("%02d", n)
}

func (s *DashboardService) GetPunctuality(ctx context.Context, branchID *int) (*DashboardPunctualityResponse, error) {
	now := time.Now()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := todayStart.Add(24 * time.Hour)

	items, err := s.getTodayPunctuality(ctx, branchID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}

	return &DashboardPunctualityResponse{
		BranchID: branchID,
		Date:     todayStart.Format("2006-01-02"),
		Items:    items,
		Total:    len(items),
	}, nil
}

func (s *DashboardService) getTodayPunctuality(ctx context.Context, branchID *int, todayStart, todayEnd time.Time) ([]TodayPunctualityItem, error) {
	args := []any{todayStart, todayEnd}
	branchWhere := buildBranchFilter("ad.branch_id", branchID, &args)

	query := fmt.Sprintf(`
		WITH today_override AS (
			SELECT DISTINCT ON (user_id) user_id, shift_id, is_day_off
			FROM user_day_overrides
			WHERE date >= $1 AND date < $2
			ORDER BY user_id, id DESC
		),
		active_assignment AS (
			SELECT DISTINCT ON (user_id) user_id, shift_id
			FROM user_shift_assignments
			WHERE is_active = true
			  AND start_date <= $1
			  AND (end_date IS NULL OR end_date > $1)
			ORDER BY user_id, id DESC
		)
		SELECT
			u.id AS user_id,
			TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS full_name,
			ad.branch_id,
			b.name AS branch_name,
			sh.name AS shift_name,
			sh.start_time,
			sh.end_time,
			sh.break_minutes,
			ad.work_in_at,
			ad.break_out_at,
			ad.break_in_at,
			ad.work_out_at,
			CASE WHEN ad.work_in_at IS NOT NULL THEN
				ROUND(EXTRACT(EPOCH FROM (ad.work_in_at - (ad.work_date::timestamp + sh.start_time::time))) / 60)::int
			ELSE NULL END AS entry_diff_minutes,
			CASE WHEN ad.break_out_at IS NOT NULL AND ad.break_in_at IS NOT NULL THEN
				ROUND(EXTRACT(EPOCH FROM (ad.break_in_at - ad.break_out_at)) / 60)::int - sh.break_minutes
			ELSE NULL END AS break_diff_minutes,
			CASE WHEN ad.work_out_at IS NOT NULL THEN
				ROUND(EXTRACT(EPOCH FROM (ad.work_out_at - (
					ad.work_date::timestamp +
					CASE WHEN sh.crosses_midnight THEN INTERVAL '1 day' ELSE INTERVAL '0' END +
					sh.end_time::time
				))) / 60)::int
			ELSE NULL END AS exit_diff_minutes
		FROM attendance_days ad
		JOIN users u ON u.id = ad.user_id
		JOIN branches b ON b.id = ad.branch_id
		LEFT JOIN today_override tov ON tov.user_id = u.id
		LEFT JOIN active_assignment aa ON aa.user_id = u.id
		LEFT JOIN shifts sh ON sh.id = COALESCE(tov.shift_id, aa.shift_id) AND sh.is_active = true
		LEFT JOIN shift_days sd ON sd.shift_id = sh.id
			AND sd.weekday = EXTRACT(DOW FROM ad.work_date)::int
		WHERE ad.work_date >= $1 AND ad.work_date < $2
		  AND (tov.is_day_off IS NULL OR tov.is_day_off = false)
		  AND sh.id IS NOT NULL
		  AND (sd.is_working_day IS NULL OR sd.is_working_day = true)
		  %s
		ORDER BY ad.work_in_at DESC NULLS LAST
	`, branchWhere)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TodayPunctualityItem, 0)
	for rows.Next() {
		var it TodayPunctualityItem
		var workInAt, breakOutAt, breakInAt, workOutAt sql.NullTime
		var entryDiff, breakDiff, exitDiff sql.NullInt64

		if err := rows.Scan(
			&it.UserID,
			&it.Name,
			&it.BranchID,
			&it.BranchName,
			&it.ShiftName,
			&it.StartTime,
			&it.EndTime,
			&it.BreakMinutes,
			&workInAt,
			&breakOutAt,
			&breakInAt,
			&workOutAt,
			&entryDiff,
			&breakDiff,
			&exitDiff,
		); err != nil {
			return nil, err
		}

		if workInAt.Valid {
			t := workInAt.Time
			it.WorkInAt = &t
		}
		if breakOutAt.Valid {
			t := breakOutAt.Time
			it.BreakOutAt = &t
		}
		if breakInAt.Valid {
			t := breakInAt.Time
			it.BreakInAt = &t
		}
		if workOutAt.Valid {
			t := workOutAt.Time
			it.WorkOutAt = &t
		}

		if entryDiff.Valid {
			v := int(entryDiff.Int64)
			it.EntryDiffMinutes = &v
		}
		if breakDiff.Valid {
			v := int(breakDiff.Int64)
			it.BreakDiffMinutes = &v
		}
		if exitDiff.Valid {
			v := int(exitDiff.Int64)
			it.ExitDiffMinutes = &v
		}

		// Entry status
		if it.WorkInAt == nil {
			it.EntryStatus = "no_mark"
		} else if *it.EntryDiffMinutes > 0 {
			it.EntryStatus = "late"
		} else if *it.EntryDiffMinutes < 0 {
			it.EntryStatus = "early"
		} else {
			it.EntryStatus = "on_time"
		}

		// Break status
		if it.BreakDiffMinutes == nil {
			it.BreakStatus = "no_break"
		} else if *it.BreakDiffMinutes > 0 {
			it.BreakStatus = "over"
		} else if *it.BreakDiffMinutes < 0 {
			it.BreakStatus = "early"
		} else {
			it.BreakStatus = "on_time"
		}

		// Exit status
		if it.WorkOutAt == nil {
			it.ExitStatus = "no_mark"
		} else if *it.ExitDiffMinutes > 0 {
			it.ExitStatus = "overtime"
		} else if *it.ExitDiffMinutes < 0 {
			it.ExitStatus = "early"
		} else {
			it.ExitStatus = "on_time"
		}

		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}