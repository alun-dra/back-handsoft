package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"math"
	"strings"
	"time"

	"back/internal/ent"
	"back/internal/ent/attendanceday"
)

var ErrInvalidMarkingsRange = errors.New("invalid markings range")
var ErrMarkingNotFound = errors.New("marking not found")
var ErrInvalidMarkingUpdate = errors.New("invalid marking update")

type MarkingsService struct {
	client *ent.Client
	db     *sql.DB
}

func NewMarkingsService(client *ent.Client, db *sql.DB) *MarkingsService {
	return &MarkingsService{client: client, db: db}
}

type MarkingsFilters struct {
	Range         string
	StartDate     *time.Time
	EndDate       *time.Time
	BranchID      *int
	AccessPointID *int
	Search        string
	Page          int
	Limit         int
}

type MarkingsFiltersResponse struct {
	Range         string `json:"range"`
	StartDate     string `json:"start_date,omitempty"`
	EndDate       string `json:"end_date,omitempty"`
	BranchID      *int   `json:"branch_id"`
	AccessPointID *int   `json:"access_point_id"`
	Search        string `json:"search"`
}

type MarkingsPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type MarkingsSummary struct {
	TotalMarkings  int `json:"total_markings"`
	LateCount      int `json:"late_count"`
	AvgLateMinutes int `json:"avg_late_minutes"`
	PeopleInside   int `json:"people_inside"`
	OvertimeCount  int `json:"overtime_count"`
}

type MarkingItem struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	Name           string     `json:"name"`
	BranchID       int        `json:"branch_id"`
	BranchName     string     `json:"branch_name"`
	AccessPointID  *int       `json:"access_point_id"`
	AccessPointName *string   `json:"access_point_name"`
	WorkDate       string     `json:"work_date"`
	WorkInAt       *time.Time `json:"work_in_at"`
	WorkOutAt      *time.Time `json:"work_out_at"`
	BreakOutAt     *time.Time `json:"break_out_at"`
	BreakInAt      *time.Time `json:"break_in_at"`
	LateMinutes    int        `json:"late_minutes"`
	EntryDiff      int        `json:"entry_diff_minutes"`
	BreakDiff      *int       `json:"break_diff_minutes"`
	ExitDiff       int        `json:"exit_diff_minutes"`
	EntryStatus    string     `json:"entry_status"`
	BreakStatus    string     `json:"break_status"`
	ExitStatus     string     `json:"exit_status"`
	HasOvertime    bool       `json:"has_overtime"`
	OvertimeMins   int        `json:"overtime_minutes"`
	EarlyExitMins  int        `json:"early_exit_minutes"`
	NetMinutes     int        `json:"net_minutes_balance"`
	NetStatus      string     `json:"net_status"`
	Edited         bool       `json:"edited"`
	LastEditReason *string    `json:"last_edit_reason"`
}

type MarkingsListResponse struct {
	Filters    MarkingsFiltersResponse `json:"filters"`
	Pagination MarkingsPagination      `json:"pagination"`
	Summary    MarkingsSummary         `json:"summary"`
	Items      []MarkingItem           `json:"items"`
}

type MarkingsFilterOption struct {
	ID       int    `json:"id"`
	BranchID *int   `json:"branch_id,omitempty"`
	Name     string `json:"name"`
}

type MarkingsFiltersOptionsResponse struct {
	Branches     []MarkingsFilterOption `json:"branches"`
	AccessPoints []MarkingsFilterOption `json:"access_points"`
}

type TopBranchItem struct {
	BranchID   int    `json:"branch_id"`
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Percentage int    `json:"percentage"`
}

type MarkingsTopBranchesResponse struct {
	Items []TopBranchItem `json:"items"`
}

type UpdateMarkingInput struct {
	WorkInAt      *string
	WorkOutAt     *string
	BreakOutAt    *string
	BreakInAt     *string
	Justification string
}

type shiftSchedule struct {
	StartTime      string
	EndTime        string
	CrossesMidnight bool
	BreakMinutes   int
}

func parseHHMM(hhmm string) (int, int, error) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

func toShiftDateTime(workDate time.Time, hhmm string, shift *shiftSchedule) (time.Time, error) {
	h, m, err := parseHHMM(hhmm)
	if err != nil {
		return time.Time{}, err
	}

	mark := time.Date(workDate.Year(), workDate.Month(), workDate.Day(), h, m, 0, 0, workDate.Location())
	if shift != nil && shift.CrossesMidnight {
		sh, sm, err := parseHHMM(shift.StartTime)
		if err == nil {
			startMin := sh*60 + sm
			markMin := h*60 + m
			if markMin < startMin {
				mark = mark.Add(24 * time.Hour)
			}
		}
	}

	return mark, nil
}

func (s *MarkingsService) getShiftSchedule(ctx context.Context, userID int, workDate time.Time) (*shiftSchedule, error) {
	overrideQuery := `
		SELECT sh.start_time, sh.end_time, sh.crosses_midnight, sh.break_minutes
		FROM user_day_overrides udo
		JOIN shifts sh ON sh.id = udo.shift_id
		WHERE udo.user_id = $1
		  AND udo.date = $2
		  AND udo.is_day_off = false
		  AND udo.shift_id IS NOT NULL
		LIMIT 1
	`

	var out shiftSchedule
	err := s.db.QueryRowContext(ctx, overrideQuery, userID, workDate).Scan(&out.StartTime, &out.EndTime, &out.CrossesMidnight, &out.BreakMinutes)
	if err == nil {
		return &out, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	query := `
		SELECT sh.start_time, sh.end_time, sh.crosses_midnight, sh.break_minutes
		FROM user_shift_assignments usa
		JOIN shifts sh ON sh.id = usa.shift_id
		WHERE usa.user_id = $1
		  AND usa.is_active = true
		  AND usa.start_date <= $2
		  AND (usa.end_date IS NULL OR usa.end_date >= $2)
		ORDER BY usa.start_date DESC, usa.id DESC
		LIMIT 1
	`

	err = s.db.QueryRowContext(ctx, query, userID, workDate).Scan(&out.StartTime, &out.EndTime, &out.CrossesMidnight, &out.BreakMinutes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &out, nil
}

func resolveMarkingsRange(r string, startDate, endDate *time.Time) (time.Time, time.Time, error) {
	if startDate != nil && endDate != nil {
		return *startDate, endDate.Add(24 * time.Hour), nil
	}

	now := time.Now()
	loc := now.Location()

	switch r {
	case "", "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return start, start.Add(24 * time.Hour), nil
	case "last_7_days":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
		start := end.AddDate(0, 0, -7)
		return start, end, nil
	case "current_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
		return start, end, nil
	case "custom":
		return time.Time{}, time.Time{}, ErrInvalidMarkingsRange
	default:
		return time.Time{}, time.Time{}, ErrInvalidMarkingsRange
	}
}

func buildMarkingsWhere(baseArgs []any, f MarkingsFilters, start, end time.Time) (string, []any) {
	args := append(baseArgs, start, end)
	where := " WHERE ad.work_date >= $1 AND ad.work_date < $2"

	if f.BranchID != nil {
		args = append(args, *f.BranchID)
		where += fmt.Sprintf(" AND ad.branch_id = $%d", len(args))
	}
	if f.AccessPointID != nil {
		args = append(args, *f.AccessPointID)
		where += fmt.Sprintf(" AND ad.access_point_id = $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where += fmt.Sprintf(" AND (LOWER(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) LIKE LOWER($%d) OR LOWER(COALESCE(u.username, '')) LIKE LOWER($%d))", len(args), len(args))
	}

	return where, args
}

func (s *MarkingsService) List(ctx context.Context, f MarkingsFilters) (*MarkingsListResponse, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 5
	}
	if f.Limit > 200 {
		f.Limit = 200
	}

	if f.Range == "" {
		f.Range = "today"
	}

	start, end, err := resolveMarkingsRange(f.Range, f.StartDate, f.EndDate)
	if err != nil {
		return nil, err
	}

	where, args := buildMarkingsWhere(nil, f, start, end)

	countQuery := "SELECT COUNT(*) FROM attendance_days ad JOIN users u ON u.id = ad.user_id" + where
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	summaryQuery := `
		SELECT
			COALESCE(SUM(
				(CASE WHEN ad.work_in_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.break_out_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.break_in_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.work_out_at IS NOT NULL THEN 1 ELSE 0 END)
			), 0) AS total_markings,
			COALESCE(SUM(CASE WHEN ad.late_minutes IS NOT NULL AND ad.late_minutes > 0 THEN 1 ELSE 0 END), 0) AS late_count,
			COALESCE(ROUND(AVG(NULLIF(ad.late_minutes, 0))), 0)::int AS avg_late_minutes,
			COALESCE(SUM(CASE WHEN ad.work_in_at IS NOT NULL AND ad.work_out_at IS NULL THEN 1 ELSE 0 END), 0) AS people_inside,
			COALESCE(SUM(CASE WHEN ad.overtime_minutes IS NOT NULL AND ad.overtime_minutes > 0 THEN 1 ELSE 0 END), 0) AS overtime_count
		FROM attendance_days ad
		JOIN users u ON u.id = ad.user_id
	` + where

	var summary MarkingsSummary
	if err := s.db.QueryRowContext(ctx, summaryQuery, args...).Scan(
		&summary.TotalMarkings,
		&summary.LateCount,
		&summary.AvgLateMinutes,
		&summary.PeopleInside,
		&summary.OvertimeCount,
	); err != nil {
		return nil, err
	}

	offset := (f.Page - 1) * f.Limit
	itemsArgs := append(args, f.Limit, offset)

	itemsQuery := fmt.Sprintf(`
		SELECT
			ad.id,
			u.id,
			TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS full_name,
			ad.branch_id,
			b.name,
			ad.access_point_id,
			ap.name,
			ad.work_date,
			ad.work_in_at,
			ad.work_out_at,
			ad.break_out_at,
			ad.break_in_at,
			COALESCE(ad.late_minutes, 0) AS late_minutes,
			CASE
				WHEN ad.break_diff_minutes IS NOT NULL THEN ad.break_diff_minutes
				WHEN ad.break_out_at IS NOT NULL AND ad.break_in_at IS NOT NULL THEN
					ROUND(EXTRACT(EPOCH FROM (ad.break_in_at - ad.break_out_at)) / 60)::int - COALESCE(sh.break_minutes, 0)
			ELSE NULL END AS break_diff,
			COALESCE(ad.overtime_minutes, 0) AS overtime_minutes,
			COALESCE(ad.early_exit_minutes, 0) AS early_exit_minutes,
			COALESCE(
				ad.net_minutes_balance,
				COALESCE(ad.overtime_minutes, 0) -
				COALESCE(ad.late_minutes, 0) -
				COALESCE(ad.early_exit_minutes, 0) -
				COALESCE(
					ad.break_diff_minutes,
					CASE
						WHEN ad.break_out_at IS NOT NULL AND ad.break_in_at IS NOT NULL THEN ROUND(EXTRACT(EPOCH FROM (ad.break_in_at - ad.break_out_at)) / 60)::int - COALESCE(sh.break_minutes, 0)
						ELSE 0
					END
				)
			) AS net_minutes_balance,
			ad.edited,
			ad.last_edit_reason
		FROM attendance_days ad
		JOIN users u ON u.id = ad.user_id
		JOIN branches b ON b.id = ad.branch_id
		LEFT JOIN access_points ap ON ap.id = ad.access_point_id
		LEFT JOIN LATERAL (
			SELECT udo.shift_id
			FROM user_day_overrides udo
			WHERE udo.user_id = ad.user_id
			  AND udo.date = ad.work_date
			  AND udo.is_day_off = false
			  AND udo.shift_id IS NOT NULL
			LIMIT 1
		) udo ON true
		LEFT JOIN LATERAL (
			SELECT usa.shift_id
			FROM user_shift_assignments usa
			WHERE usa.user_id = ad.user_id
			  AND usa.is_active = true
			  AND usa.start_date <= ad.work_date
			  AND (usa.end_date IS NULL OR usa.end_date >= ad.work_date)
			ORDER BY usa.start_date DESC, usa.id DESC
			LIMIT 1
		) usa ON true
		LEFT JOIN shifts sh ON sh.id = COALESCE(udo.shift_id, usa.shift_id)
		%s
		ORDER BY COALESCE(ad.work_in_at, ad.work_out_at, ad.updated_at) DESC
		LIMIT $%d OFFSET $%d
	`, where, len(itemsArgs)-1, len(itemsArgs))

	rows, err := s.db.QueryContext(ctx, itemsQuery, itemsArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MarkingItem, 0)
	for rows.Next() {
		var it MarkingItem
		var workDate time.Time
		var workIn, workOut, breakOut, breakIn sql.NullTime
		var accessPointName sql.NullString
		var lastEditReason sql.NullString
		var breakDiff sql.NullInt64

		if err := rows.Scan(
			&it.ID,
			&it.UserID,
			&it.Name,
			&it.BranchID,
			&it.BranchName,
			&it.AccessPointID,
			&accessPointName,
			&workDate,
			&workIn,
			&workOut,
			&breakOut,
			&breakIn,
			&it.LateMinutes,
			&breakDiff,
			&it.OvertimeMins,
			&it.EarlyExitMins,
			&it.NetMinutes,
			&it.Edited,
			&lastEditReason,
		); err != nil {
			return nil, err
		}

		it.WorkDate = workDate.Format("2006-01-02")
		if accessPointName.Valid {
			v := accessPointName.String
			it.AccessPointName = &v
		}
		if workIn.Valid {
			t := workIn.Time
			it.WorkInAt = &t
		}
		if workOut.Valid {
			t := workOut.Time
			it.WorkOutAt = &t
		}
		if breakOut.Valid {
			t := breakOut.Time
			it.BreakOutAt = &t
		}
		if breakIn.Valid {
			t := breakIn.Time
			it.BreakInAt = &t
		}
		if breakDiff.Valid {
			v := int(breakDiff.Int64)
			it.BreakDiff = &v
		}
		if lastEditReason.Valid {
			v := lastEditReason.String
			it.LastEditReason = &v
		}

		it.EntryDiff = it.LateMinutes
		if it.OvertimeMins > 0 {
			it.ExitDiff = it.OvertimeMins
		} else if it.EarlyExitMins > 0 {
			it.ExitDiff = -it.EarlyExitMins
		} else {
			it.ExitDiff = 0
		}

		switch {
		case it.NetMinutes > 0:
			it.NetStatus = "positive"
		case it.NetMinutes < 0:
			it.NetStatus = "negative"
		default:
			it.NetStatus = "neutral"
		}

		if it.WorkInAt == nil {
			it.EntryStatus = "no_mark"
		} else if it.EntryDiff > 0 {
			it.EntryStatus = "late"
		} else {
			it.EntryStatus = "ok"
		}

		if it.BreakDiff == nil {
			it.BreakStatus = "no_mark"
		} else if *it.BreakDiff > 0 {
			it.BreakStatus = "over"
		} else if *it.BreakDiff < 0 {
			it.BreakStatus = "early"
		} else {
			it.BreakStatus = "ok"
		}

		if it.WorkOutAt == nil {
			it.ExitStatus = "no_mark"
		} else if it.ExitDiff > 0 {
			it.ExitStatus = "overtime"
		} else if it.ExitDiff < 0 {
			it.ExitStatus = "early"
		} else {
			it.ExitStatus = "ok"
		}

		it.HasOvertime = it.OvertimeMins > 0

		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(f.Limit)))
	}

	resp := &MarkingsListResponse{
		Filters: MarkingsFiltersResponse{
			Range:         f.Range,
			StartDate:     start.Format("2006-01-02"),
			EndDate:       end.Add(-time.Nanosecond).Format("2006-01-02"),
			BranchID:      f.BranchID,
			AccessPointID: f.AccessPointID,
			Search:        f.Search,
		},
		Pagination: MarkingsPagination{
			Page:       f.Page,
			Limit:      f.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
		Summary: summary,
		Items:   items,
	}
	if resp.Items == nil {
		resp.Items = []MarkingItem{}
	}
	return resp, nil
}

func (s *MarkingsService) Update(ctx context.Context, id int, in UpdateMarkingInput) (*MarkingItem, error) {
	justification := strings.TrimSpace(in.Justification)
	if justification == "" {
		return nil, ErrInvalidMarkingUpdate
	}

	ad, err := s.client.AttendanceDay.Query().Where(attendanceday.IDEQ(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarkingNotFound
		}
		return nil, err
	}

	shift, err := s.getShiftSchedule(ctx, ad.UserID, ad.WorkDate)
	if err != nil {
		return nil, err
	}

	update := s.client.AttendanceDay.UpdateOneID(id)

	workIn := ad.WorkInAt
	if in.WorkInAt != nil {
		v := strings.TrimSpace(*in.WorkInAt)
		if v == "" {
			update.ClearWorkInAt()
			workIn = nil
		} else {
			t, parseErr := toShiftDateTime(ad.WorkDate, v, shift)
			if parseErr != nil {
				return nil, ErrInvalidMarkingUpdate
			}
			update.SetWorkInAt(t)
			workIn = &t
		}
	}

	workOut := ad.WorkOutAt
	if in.WorkOutAt != nil {
		v := strings.TrimSpace(*in.WorkOutAt)
		if v == "" {
			update.ClearWorkOutAt()
			workOut = nil
		} else {
			t, parseErr := toShiftDateTime(ad.WorkDate, v, shift)
			if parseErr != nil {
				return nil, ErrInvalidMarkingUpdate
			}
			update.SetWorkOutAt(t)
			workOut = &t
		}
	}

	breakOut := ad.BreakOutAt

	if in.BreakOutAt != nil {
		v := strings.TrimSpace(*in.BreakOutAt)
		if v == "" {
			update.ClearBreakOutAt()
			breakOut = nil
		} else {
			t, parseErr := toShiftDateTime(ad.WorkDate, v, shift)
			if parseErr != nil {
				return nil, ErrInvalidMarkingUpdate
			}
			update.SetBreakOutAt(t)
			breakOut = &t
		}
	}

	breakIn := ad.BreakInAt

	if in.BreakInAt != nil {
		v := strings.TrimSpace(*in.BreakInAt)
		if v == "" {
			update.ClearBreakInAt()
			breakIn = nil
		} else {
			t, parseErr := toShiftDateTime(ad.WorkDate, v, shift)
			if parseErr != nil {
				return nil, ErrInvalidMarkingUpdate
			}
			update.SetBreakInAt(t)
			breakIn = &t
		}
	}

	metrics := computeAttendanceMetrics(ad.WorkDate, attendanceMetricsSchedule{
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		CrossesMidnight: shift.CrossesMidnight,
		BreakMinutes:    shift.BreakMinutes,
	}, workIn, breakOut, breakIn, workOut)

	if metrics.LateMinutes != nil {
		update.SetLateMinutes(*metrics.LateMinutes)
	} else {
		update.ClearLateMinutes()
	}
	if metrics.BreakDiffMinutes != nil {
		update.SetBreakDiffMinutes(*metrics.BreakDiffMinutes)
	} else {
		update.ClearBreakDiffMinutes()
	}
	if metrics.OvertimeMinutes != nil {
		update.SetOvertimeMinutes(*metrics.OvertimeMinutes)
	} else {
		update.ClearOvertimeMinutes()
	}
	if metrics.EarlyExitMinutes != nil {
		update.SetEarlyExitMinutes(*metrics.EarlyExitMinutes)
	} else {
		update.ClearEarlyExitMinutes()
	}
	if metrics.NetMinutes != nil {
		update.SetNetMinutesBalance(*metrics.NetMinutes)
	} else {
		update.ClearNetMinutesBalance()
	}

	now := time.Now()
	update.SetEdited(true)
	update.SetLastEditReason(justification)
	update.SetEditedAt(now)

	if _, err := update.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarkingNotFound
		}
		return nil, err
	}

	resp, err := s.List(ctx, MarkingsFilters{
		Range: "custom",
		StartDate: &ad.WorkDate,
		EndDate: &ad.WorkDate,
		Page: 1,
		Limit: 500,
	})
	if err != nil {
		return nil, err
	}

	for i := range resp.Items {
		if resp.Items[i].ID == id {
			item := resp.Items[i]
			return &item, nil
		}
	}

	return nil, ErrMarkingNotFound
}

func (s *MarkingsService) Filters(ctx context.Context) (*MarkingsFiltersOptionsResponse, error) {
	branchesRows, err := s.db.QueryContext(ctx, `
		SELECT id, name
		FROM branches
		WHERE is_active = true
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer branchesRows.Close()

	branches := make([]MarkingsFilterOption, 0)
	for branchesRows.Next() {
		var b MarkingsFilterOption
		if err := branchesRows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	if err := branchesRows.Err(); err != nil {
		return nil, err
	}

	accessRows, err := s.db.QueryContext(ctx, `
		SELECT id, branch_id, name
		FROM access_points
		WHERE is_active = true
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer accessRows.Close()

	accessPoints := make([]MarkingsFilterOption, 0)
	for accessRows.Next() {
		var ap MarkingsFilterOption
		var branchID int
		if err := accessRows.Scan(&ap.ID, &branchID, &ap.Name); err != nil {
			return nil, err
		}
		ap.BranchID = &branchID
		accessPoints = append(accessPoints, ap)
	}
	if err := accessRows.Err(); err != nil {
		return nil, err
	}

	if branches == nil {
		branches = []MarkingsFilterOption{}
	}
	if accessPoints == nil {
		accessPoints = []MarkingsFilterOption{}
	}

	return &MarkingsFiltersOptionsResponse{Branches: branches, AccessPoints: accessPoints}, nil
}

func (s *MarkingsService) TopBranches(ctx context.Context, f MarkingsFilters) (*MarkingsTopBranchesResponse, error) {
	if f.Range == "" {
		f.Range = "today"
	}
	start, end, err := resolveMarkingsRange(f.Range, f.StartDate, f.EndDate)
	if err != nil {
		return nil, err
	}

	args := []any{start, end}
	where := " WHERE ad.work_date >= $1 AND ad.work_date < $2"
	if f.BranchID != nil {
		args = append(args, *f.BranchID)
		where += fmt.Sprintf(" AND ad.branch_id = $%d", len(args))
	}

	query := `
		SELECT
			b.id,
			b.name,
			COALESCE(SUM(
				(CASE WHEN ad.work_in_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.break_out_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.break_in_at IS NOT NULL THEN 1 ELSE 0 END) +
				(CASE WHEN ad.work_out_at IS NOT NULL THEN 1 ELSE 0 END)
			), 0) AS cnt
		FROM branches b
		LEFT JOIN attendance_days ad ON ad.branch_id = b.id
	`
	query += where
	query += `
		GROUP BY b.id, b.name
		HAVING COALESCE(SUM(
			(CASE WHEN ad.work_in_at IS NOT NULL THEN 1 ELSE 0 END) +
			(CASE WHEN ad.break_out_at IS NOT NULL THEN 1 ELSE 0 END) +
			(CASE WHEN ad.break_in_at IS NOT NULL THEN 1 ELSE 0 END) +
			(CASE WHEN ad.work_out_at IS NOT NULL THEN 1 ELSE 0 END)
		), 0) > 0
		ORDER BY cnt DESC, b.name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TopBranchItem, 0)
	total := 0
	for rows.Next() {
		var it TopBranchItem
		if err := rows.Scan(&it.BranchID, &it.Name, &it.Count); err != nil {
			return nil, err
		}
		total += it.Count
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if total > 0 {
		for i := range items {
			items[i].Percentage = int(math.Round((float64(items[i].Count) / float64(total)) * 100.0))
		}
	}

	if items == nil {
		items = []TopBranchItem{}
	}

	return &MarkingsTopBranchesResponse{Items: items}, nil
}
