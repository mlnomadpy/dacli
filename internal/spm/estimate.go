package spm

import "fmt"

// ThreePoint is a PERT estimate. dacli refuses scalar estimates on purpose:
// an agent asked for a number produces a confident point value with no error
// bars, every time. Forcing three points forces the pessimistic case to be
// stated, and the pessimistic case is where the unexamined risk lives.
type ThreePoint struct {
	Optimistic  float64 `yaml:"optimistic"`
	Probable    float64 `yaml:"probable"`
	Pessimistic float64 `yaml:"pessimistic"`
}

func (t ThreePoint) Valid() error {
	switch {
	case t.Optimistic < 0:
		return fmt.Errorf("optimistic estimate is negative")
	case t.Optimistic > t.Probable:
		return fmt.Errorf("optimistic (%g) exceeds probable (%g)", t.Optimistic, t.Probable)
	case t.Probable > t.Pessimistic:
		return fmt.Errorf("probable (%g) exceeds pessimistic (%g)", t.Probable, t.Pessimistic)
	}
	return nil
}

// Expected is the PERT weighted average: Te = (To + 4Tm + Tp) / 6.
func (t ThreePoint) Expected() float64 {
	return (t.Optimistic + 4*t.Probable + t.Pessimistic) / 6
}

// Sigma is the PERT standard deviation: σ = (Tp − To) / 6.
func (t ThreePoint) Sigma() float64 {
	return (t.Pessimistic - t.Optimistic) / 6
}

// Triangular is the alternative average, weighting the most probable value
// less: Te = (To + Tm + Tp) / 3. Use when the probable value is itself a guess.
func (t ThreePoint) Triangular() float64 {
	return (t.Optimistic + t.Probable + t.Pessimistic) / 3
}

// ConfidenceRange returns the interval at the given number of standard
// deviations: 1σ ≈ 68.3% confidence, 2σ ≈ 95.5%.
func (t ThreePoint) ConfidenceRange(sigmas float64) (low, high float64) {
	te, s := t.Expected(), t.Sigma()
	low = te - sigmas*s
	if low < 0 {
		low = 0
	}
	return low, te + sigmas*s
}

// Stage is where a project sits on the Cone of Uncertainty. Estimates made
// early are not merely uncertain, they are uncertain by a known multiple, and
// reporting a point value at StageDefinition is close to a lie.
type Stage string

const (
	StageDefinition  Stage = "definition"  // initial product definition
	StageElicitation Stage = "elicitation" // requirements elicited
	StageApproach    Stage = "approach"    // approach formulated
	StageDesign      Stage = "design"      // architecture / detailed design
)

// cone holds McConnell's variability factors (Software Estimation, 2006).
var cone = map[Stage][2]float64{
	StageDefinition:  {0.375, 3.0},
	StageElicitation: {0.5, 2.0},
	StageApproach:    {0.8, 1.25},
	StageDesign:      {0.9, 1.1},
}

// ConeRange applies the Cone of Uncertainty to an expected value.
// A 6-unit estimate at elicitation reports as 3–12, which is the honest answer.
func ConeRange(expected float64, s Stage) (low, high float64) {
	f, ok := cone[s]
	if !ok {
		return expected, expected
	}
	return expected * f[0], expected * f[1]
}

// Estimate is a full report for one task.
type Estimate struct {
	Points     ThreePoint
	Stage      Stage
	Expected   float64
	Sigma      float64
	ConeLow    float64
	ConeHigh   float64
	Sigma1Low  float64
	Sigma1High float64
}

// Evaluate computes the full estimate report.
func Evaluate(t ThreePoint, s Stage) (Estimate, error) {
	if err := t.Valid(); err != nil {
		return Estimate{}, err
	}
	e := Estimate{Points: t, Stage: s, Expected: t.Expected(), Sigma: t.Sigma()}
	e.ConeLow, e.ConeHigh = ConeRange(e.Expected, s)
	e.Sigma1Low, e.Sigma1High = t.ConfidenceRange(1)
	return e, nil
}

// IsEpic reports whether an estimate is too large to be a single task.
//
// The INVEST "Small" criterion, in agent terms: a task whose pessimistic case
// exceeds one session's budget cannot be finished in one session, so it will
// be abandoned mid-flight and picked up by an agent with no memory of it.
// Decompose it instead.
func (t ThreePoint) IsEpic(sessionBudget float64) bool {
	return sessionBudget > 0 && t.Pessimistic > sessionBudget
}
