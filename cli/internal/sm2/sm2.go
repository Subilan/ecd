package sm2

import "math"

const (
	MinEF           = 1.3
	EasyBonus       = 1.3
	HardMultiplier  = 1.2
	HardEFPenalty   = 0.15
	EasyEFBonus     = 0.15
)

// ReviewOutcome maps button presses to SM-2 quality.
type ReviewOutcome int

const (
	Again ReviewOutcome = iota // 0
	Hard                       // 1
	Good                       // 2
	Easy                       // 3
)

// SM2Params holds the scheduling state for a flashcard.
type SM2Params struct {
	Repetitions  int
	IntervalDays int
	EaseFactor   float64
}

// Schedule calculates new SM-2 scheduling values after a review.
// button: 0=Again (forgot), 1=Hard, 2=Good, 3=Easy
func Schedule(outcome ReviewOutcome, p SM2Params) SM2Params {
	qMap := map[ReviewOutcome]int{Again: 0, Hard: 2, Good: 4, Easy: 5}
	q := qMap[outcome]

	ef := p.EaseFactor + (0.1 - float64(5-q)*(0.08+float64(5-q)*0.02))
	ef = math.Max(MinEF, math.Round(ef*100)/100)

	switch outcome {
	case Again:
		return SM2Params{Repetitions: 0, IntervalDays: 0, EaseFactor: ef}

	case Hard:
		newInt := 1
		if p.IntervalDays > 0 {
			newInt = int(math.Max(1, math.Round(float64(p.IntervalDays)*HardMultiplier)))
		}
		ef -= HardEFPenalty
		ef = math.Max(MinEF, math.Round(ef*100)/100)
		return SM2Params{Repetitions: p.Repetitions, IntervalDays: newInt, EaseFactor: ef}

	case Good:
		newInt := 1
		if p.Repetitions == 0 {
			newInt = 1
		} else if p.Repetitions == 1 {
			newInt = 6
		} else {
			newInt = int(math.Round(float64(p.IntervalDays) * p.EaseFactor))
		}
		newInt = max(1, newInt)
		return SM2Params{Repetitions: p.Repetitions + 1, IntervalDays: newInt, EaseFactor: ef}

	case Easy:
		newInt := 1
		if p.Repetitions == 0 {
			newInt = 1
		} else if p.Repetitions == 1 {
			newInt = 6
		} else {
			newInt = int(math.Round(float64(p.IntervalDays) * p.EaseFactor))
		}
		newInt = int(math.Round(float64(newInt) * EasyBonus))
		newInt = max(1, newInt)
		ef += EasyEFBonus
		ef = math.Round(ef*100) / 100
		return SM2Params{Repetitions: p.Repetitions + 1, IntervalDays: newInt, EaseFactor: ef}

	default:
		return p
	}
}
