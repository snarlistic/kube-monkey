package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsWeekDay(t *testing.T) {
	monday := time.Date(2018, 4, 16, 0, 0, 0, 0, time.UTC)

	assert.True(t, isWeekday(monday))
	assert.True(t, isWeekday(monday.Add(time.Hour*24)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*2)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*3)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*4)))

	assert.False(t, isWeekday(monday.Add(time.Hour*24*5)))
	assert.False(t, isWeekday(monday.Add(time.Hour*24*6)))
}

func TestShouldParseMtbf(t *testing.T) {
	duration, _ := ParseMtbf("2d")
	assert.Equal(t, time.Hour*24*2, duration)

	duration, _ = ParseMtbf("2h")
	assert.Equal(t, time.Hour*2, duration)

	duration, _ = ParseMtbf("2m")
	assert.Equal(t, time.Minute*2, duration)

	// Special case where we don't have a time unit and we assume days
	duration, _ = ParseMtbf("2")
	assert.Equal(t, duration, time.Hour*24*2)
}

func TestParseMtbfShouldNotAllowDurationLessThanAMinute(t *testing.T) {
	_, err := ParseMtbf("30s")
	assert.NotNil(t, err)
}

// FIXME:  add more tests
