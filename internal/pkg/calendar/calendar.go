package calendar

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

// Checks if specified Time is a weekday
func isWeekday(t time.Time) bool {
	switch t.Weekday() {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday:
		return true
	case time.Saturday, time.Sunday:
		return false
	}

	glog.Fatalf("Unrecognized day of the week: %s", t.Weekday().String())

	panic("Explicit Panic to avoid compiler error: missing return at end of function")
}

// Returns the next weekday in Location
func nextWeekday(loc *time.Location) time.Time {
	check := time.Now().In(loc)
	for {
		check = check.AddDate(0, 0, 1)
		if isWeekday(check) {
			return check
		}
	}
}

// NextRuntime calculates the next time the Scheduled should run
func NextRuntime(loc *time.Location, r int) time.Time {
	now := time.Now().In(loc)

	// Is today a weekday and are we still in time for it?
	if isWeekday(now) {
		runtimeToday := time.Date(now.Year(), now.Month(), now.Day(), r, 0, 0, 0, loc)
		if runtimeToday.After(now) {
			return runtimeToday
		}
	}

	// Missed the train for today. Schedule on next weekday
	year, month, day := nextWeekday(loc).Date()
	return time.Date(year, month, day, r, 0, 0, 0, loc)
}

// ParseMtbf parses an mtbf value and returns a valid time duration.
func ParseMtbf(mtbf string) (time.Duration, error) {
	// time.Duration biggest valid time unit is an hour, but we want to accept
	// days. Before finer grained time units this software used to accept mtbf as
	// an integer interpreted as days. Hence this routine now accepts a "d" as a
	// valid time unit meaning days and simply strips it, because...
	if mtbf[len(mtbf) - 1] == 'd' {
		mtbf = strings.TrimRight(mtbf, "d")
	}
	// ...below we check if a given mtbf is simply a number and backward
	// compatibilty dictates us to accept a simpel number as days (see above) and
	// since time.Duration does not accept hours as a valid time unit we convert
	// here ourselves days into hours.
	if converted_mtbf, err := strconv.Atoi(mtbf); err == nil {
		mtbf = fmt.Sprintf("%dh", converted_mtbf * 24)
	}
	duration, err := time.ParseDuration(mtbf)
	if err != nil {
		return 0, err
	}
	one_minute, _ := time.ParseDuration("1m")
	if duration < one_minute {
		return 0, errors.New("smallest valid mtbf is one minute.")
	}
	return duration, nil
}

// RandomOffsetTime returns a random time offset from the given start time within the given time window
func RandomOffsetTime(r *rand.Rand, startTime time.Time, timeWindow time.Duration) time.Time {
	// Convert the time window into seconds and get a random number of seconds within that window
	subSecond := int64(timeWindow / time.Second)
	randSecondOffset := r.Int63n(subSecond)
	// Add the random time offset to the start time before returning it to the caller
	randCalTime := startTime.Add(time.Duration(randSecondOffset) * time.Second)
	return randCalTime
}

// RandomTimeInRange returns a slice of random times within the range specified by startHour and endHour
func RandomTimeInRange(mtbf string, startHour int, endHour int, loc *time.Location) []time.Time {
	var times []time.Time

	// Parse the mtbf (mean time between failure) and convert it into a time
	// duration that should be twice the length of the mtbf. It is an average
	// after all.
	tmptimeDuration, err := ParseMtbf(mtbf)
	if err != nil {
		glog.Errorf("error parsing customized mtbf %s: %v", mtbf, err)
		return []time.Time{time.Now().Add(time.Duration(24*365*10) * time.Hour)}
	}
	timeDuration := tmptimeDuration * 2

	// Initialize (seed) a random number generator and deduce the current start
	// time of schedule creation for future reference.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	startTime := time.Now().In(loc)

	// Deduce the time window of operation.
	year, month, date := startTime.Date()
	todayStartTime := time.Date(year, month, date, startHour, 0, 0, 0, loc)
	todayEndTime := time.Date(year, month, date, endHour, 0, 0, 0, loc)
	timeWindow := todayEndTime.Sub(todayStartTime)

	// If the time duration (mbtf * 2) is bigger than the available time window
	// the full 24 hour day is used to calculate the probability. If only using
	// the time window it will result in incorrect probability. In this case only
	// one time calculation is needed as the mtbf is bigger than the available
	// time window resulting in only one occurence within the time window if
	// probability makes it occur anyway.
	//
	// If the time duration is smaller than the time window it will fit at least
	// one but most probably many times within it. For that reason  the
	// occurances are calculated in a tiny different fashion, making sure they
	// each and everyone will fit within the time window.
	//
	// Since both cases need at least one run this is exactly what the above
	// loop is doing.
	for again := true; again; {
		// Is the time window bigger than the time duration? If so this loop
		// will run again and again.
		timeWindowBiggerThanTimeDuration := timeWindow > timeDuration
		again = timeWindowBiggerThanTimeDuration
		if timeWindowBiggerThanTimeDuration {
			// It is indeed bigger, so a random offset is calculated and added
			// to the start time of the schedule creation.
			mtbfEndTime := startTime.Add(timeDuration)
			randCalTime := RandomOffsetTime(r, startTime, mtbfEndTime.Sub(startTime))
			// Check if the new calculated time is within the time window. It must
			// be before the end of day and ...
			if randCalTime.Before(todayEndTime) {
				// ... after the start of day.
				if startTime.After(todayStartTime) {
					// Yes it is! Add it to the calculated slice of times.
					times = append(times, randCalTime)
				}
				// If is was or was not, either way move start time up to the
				// calculated random time for the next calculation.
				startTime = randCalTime
			} else {
				// The latest calculated time is after the end of day, so no
				// need to calculate more times.
				again = false
			}
		} else {
			// The time window was not bigger, so only calculate one time as
			// described above. First take a time definition of one day and divide it
			// by time duration. The will produce the probability that the failure
			// need to occur today. Next get a random number in between 0 and 1 that
			// will be used to see if the failure will occur this time or not. If it
			// is bigger than the probability it does. Does it this time?
			one_day, _ := time.ParseDuration("24h")
			if (float64(one_day) / float64(timeDuration)) > r.Float64() {
				// Yes, it does. So allow it to happen by getting a random time
				// anywhere within the time window offset by the start of today
				// and add it to the calculated slice of times.
				randCalTime := RandomOffsetTime(r, todayStartTime, timeWindow)
				times = append(times, randCalTime)
			}
		}
	}
	// Return the slice with all calculated times.
	return times
}
