package utils

import (
	"strconv"
	"strings"
	"time"

	"toentoen-backend/internal/domain/shared/models"
)

var defaultWIBLocation = time.FixedZone("WIB", 7*3600)

// GetWIBLocation returns the Asia/Jakarta (WIB / UTC+7) location, falling back to a fixed zone if tzdata is missing.
func GetWIBLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil || loc == nil {
		return defaultWIBLocation
	}
	return loc
}

// CalculateStoreStatus evaluates real-time business open status locked to WIB (UTC+7),
// accounting for manual closures, daily operating schedules, and overtime confirmations.
func CalculateStoreStatus(isManuallyClosed bool, hours []models.BusinessHour, confirmedDates ...*string) bool {
	// Immediate return if business owner triggered manual emergency closure.
	if isManuallyClosed {
		return false
	}

	// Lock evaluation to WIB (GMT+7).
	loc := GetWIBLocation()
	now := time.Now().In(loc)
	todayStr := now.Format("2006-01-02")
	currentDay := int(now.Weekday()) // 0 = Minggu, 1 = Senin, ..., 6 = Sabtu

	var confirmedOpenDate *string
	var confirmedOvertimeDate *string
	if len(confirmedDates) > 0 {
		confirmedOpenDate = confirmedDates[0]
	}
	if len(confirmedDates) > 1 {
		confirmedOvertimeDate = confirmedDates[1]
	}

	isOvertimeConfirmedToday := confirmedOvertimeDate != nil && strings.HasPrefix(strings.TrimSpace(*confirmedOvertimeDate), todayStr)
	isOpenConfirmedToday := (confirmedOpenDate != nil && strings.HasPrefix(strings.TrimSpace(*confirmedOpenDate), todayStr)) || isOvertimeConfirmedToday

	nowMinutes := now.Hour()*60 + now.Minute()

	// Evaluate today operational schedule.
	hasScheduleToday := false
	for _, h := range hours {
		if h.DayOfWeek == currentDay {
			if h.IsClosed {
				return false
			}
			if h.OpenTime != nil && h.CloseTime != nil {
				hasScheduleToday = true
				openStr := strings.Split(strings.TrimSpace(*h.OpenTime), ":")
				closeStr := strings.Split(strings.TrimSpace(*h.CloseTime), ":")

				if len(openStr) >= 2 && len(closeStr) >= 2 {
					openHour, errOH := strconv.Atoi(openStr[0])
					openMinute, errOM := strconv.Atoi(openStr[1])
					closeHour, errCH := strconv.Atoi(closeStr[0])
					closeMinute, errCM := strconv.Atoi(closeStr[1])

					if errOH == nil && errOM == nil && errCH == nil && errCM == nil {
						openTotalMinutes := openHour*60 + openMinute
						closeTotalMinutes := closeHour*60 + closeMinute

						// Before regular opening hours; open only if confirmed for early hours or overtime.
						if nowMinutes < openTotalMinutes {
							return isOvertimeConfirmedToday
						}

						// Store must have daily open confirmation for regular hours.
						if !isOpenConfirmedToday {
							return false
						}

						// 24-hour schedule.
						if openTotalMinutes == 0 && (closeTotalMinutes == 0 || closeTotalMinutes >= 1439) {
							return true
						}

						// Regular same-day schedule.
						if openTotalMinutes <= closeTotalMinutes {
							if nowMinutes <= closeTotalMinutes {
								return true
							}
							// Past regular closing time; open only if overtime is confirmed for today.
							return isOvertimeConfirmedToday
						}

						// Overnight schedule spanning midnight.
						if nowMinutes >= openTotalMinutes || nowMinutes <= closeTotalMinutes {
							return true
						}
						return isOvertimeConfirmedToday
					}
				}
			}
		}
	}

	// When no schedule is defined for today, open status defaults to manual toggle state.
	if !hasScheduleToday {
		return !isManuallyClosed
	}

	return false
}

// ParseFlexibleTime parses date-time strings across standard formats (RFC3339, SQL datetime, ISO-8601, Unix epoch) into WIB location.
func ParseFlexibleTime(val string) (time.Time, error) {
	val = strings.TrimSpace(val)
	if val == "" || val == "null" {
		return time.Time{}, nil
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999Z07:00",
		"2006-01-02T15:04:05.99Z07:00",
		"2006-01-02T15:04:05.9Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05+07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	loc := GetWIBLocation()
	for _, layout := range formats {
		if t, err := time.ParseInLocation(layout, val, loc); err == nil {
			return t, nil
		}
	}

	// Fall back to Unix epoch timestamp (seconds or milliseconds).
	if epoch, err := strconv.ParseInt(val, 10, 64); err == nil {
		if epoch > 1e11 {
			return time.UnixMilli(epoch), nil
		}
		return time.Unix(epoch, 0), nil
	}

	return time.Parse(time.RFC3339, val)
}
