package services

import "time"

type attendanceMetricsSchedule struct {
	StartTime       string
	EndTime         string
	CrossesMidnight bool
	BreakMinutes    int
}

type attendanceMetrics struct {
	LateMinutes      *int
	BreakDiffMinutes *int
	OvertimeMinutes  *int
	EarlyExitMinutes *int
	NetMinutes       *int
}

func computeAttendanceMetrics(workDate time.Time, schedule attendanceMetricsSchedule, workIn, breakOut, breakIn, workOut *time.Time) attendanceMetrics {
	metrics := attendanceMetrics{}

	startT, startErr := toShiftBoundary(workDate, schedule.StartTime, false)
	endT, endErr := toShiftBoundary(workDate, schedule.EndTime, schedule.CrossesMidnight)

	if startErr == nil && workIn != nil {
		late := int(workIn.Sub(startT).Minutes())
		if late < 0 {
			late = 0
		}
		metrics.LateMinutes = intPtr(late)
	}

	if breakOut != nil && breakIn != nil {
		breakDiff := int(breakIn.Sub(*breakOut).Minutes()) - schedule.BreakMinutes
		metrics.BreakDiffMinutes = intPtr(breakDiff)
	}

	if endErr == nil && workOut != nil {
		diff := int(workOut.Sub(endT).Minutes())
		if diff > 0 {
			metrics.OvertimeMinutes = intPtr(diff)
			metrics.EarlyExitMinutes = intPtr(0)
		} else if diff < 0 {
			metrics.OvertimeMinutes = intPtr(0)
			metrics.EarlyExitMinutes = intPtr(-diff)
		} else {
			metrics.OvertimeMinutes = intPtr(0)
			metrics.EarlyExitMinutes = intPtr(0)
		}
	}

	if metrics.LateMinutes != nil || metrics.BreakDiffMinutes != nil || metrics.OvertimeMinutes != nil || metrics.EarlyExitMinutes != nil {
		net := valueOrZero(metrics.OvertimeMinutes) - valueOrZero(metrics.LateMinutes) - valueOrZero(metrics.EarlyExitMinutes) - valueOrZero(metrics.BreakDiffMinutes)
		metrics.NetMinutes = intPtr(net)
	}

	return metrics
}

func toShiftBoundary(workDate time.Time, hhmm string, nextDay bool) (time.Time, error) {
	h, m, err := parseHHMM(hhmm)
	if err != nil {
		return time.Time{}, err
	}

	boundary := time.Date(workDate.Year(), workDate.Month(), workDate.Day(), h, m, 0, 0, workDate.Location())
	if nextDay {
		boundary = boundary.Add(24 * time.Hour)
	}
	return boundary, nil
}

func intPtr(v int) *int {
	return &v
}

func valueOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}