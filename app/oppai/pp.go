package oppai

import (
	"github.com/tsunyoku/danser/app/beatmap/difficulty"
	"github.com/tsunyoku/danser/app/bmath"
	"math"
)

/* ------------------------------------------------------------- */
/* pp calc                                                       */

/* base pp value for stars, used internally by ppv2 */
func ppBase(stars float64) float64 {
	return math.Pow(5.0*math.Max(1.0, stars/0.0675)-4.0, 3.0) /
		100000.0
}

// PPv2 : structure to store ppv2 values
type PPv2 struct {
	Total, Aim, Speed, Acc float64

	aimStrain, speedStrain float64

	maxCombo, nsliders, ncircles, nobjects int

	scoreMaxCombo int
	countGreat    int
	countOk       int
	countMeh      int
	countMiss     int

	diff *difficulty.Difficulty

	totalHits                    int
	accuracy                     float64
	amountHitObjectsWithAccuracy int
}

func (pp *PPv2) PPv2x(aimStars, speedStars float64,
	maxCombo, nsliders, ncircles, nobjects,
	combo, n300, n100, n50, nmiss int, diff *difficulty.Difficulty,
	scoreVersion int) PPv2 {
	maxCombo = bmath.MaxI(1, maxCombo)

	pp.maxCombo, pp.nsliders, pp.ncircles, pp.nobjects = maxCombo, nsliders, ncircles, nobjects

	if combo < 0 {
		combo = maxCombo
	}

	if n300 < 0 {
		n300 = nobjects - n100 - n50 - nmiss
	}

	totalhits := n300 + n100 + n50 + nmiss

	pp.aimStrain = aimStars
	pp.speedStrain = speedStars
	pp.diff = diff
	pp.totalHits = totalhits
	pp.scoreMaxCombo = combo
	pp.countGreat = n300
	pp.countOk = n100
	pp.countMeh = n50
	pp.countMiss = nmiss

	// accuracy

	if totalhits == 0 {
		pp.accuracy = 0.0
	} else {
		acc := (float64(n50)*50 +
			float64(n100)*100 +
			float64(n300)*300) /
			(float64(totalhits) * 300)

		pp.accuracy = bmath.ClampF64(acc, 0, 1)
	}

	switch scoreVersion {
	case 1:
		pp.amountHitObjectsWithAccuracy = ncircles
	case 2:
		pp.amountHitObjectsWithAccuracy = nobjects
	default:
		panic("unsupported score")
	}

	// total pp

	finalMultiplier := 1.12

	if diff.Mods.Active(difficulty.NoFail) {
		finalMultiplier *= math.Max(0.90, 1.0-0.02*float64(nmiss))
	}

	if totalhits > 0 && diff.Mods.Active(difficulty.SpunOut) {
		nspinners := nobjects - nsliders - ncircles

		finalMultiplier *= 1.0 - math.Pow(float64(nspinners)/float64(totalhits), 0.85)
	}

	aim := pp.computeAimValue()
	speed := pp.computeSpeedValue()
	accuracy := pp.computeAccuracyValue()

	if pp.diff.Mods.Active(difficulty.Relax) {
		streams_nerf := aim / speed
		if (streams_nerf < 1.0) {
			if (pp.accuracy >= 0.99) {
				aim *= 0.94
			} else if (pp.accuracy >= 0.98) {
				aim *= 0.92
			} else if (pp.accuracy >= 0.97) {
				aim *= 0.9
			} else {
				aim *= 0.87
			}
		}

		pp.Total = math.Pow(
			math.Pow(aim, 1.17) + math.Pow(accuracy, 1.15),
			1.0 / 1.1) * finalMultiplier
	} else {
		pp.Total = math.Pow(
			math.Pow(aim, 1.1)+math.Pow(speed, 1.1)+
				math.Pow(accuracy, 1.1),
			1.0/1.1) * finalMultiplier
	}

	return *pp
}

func (pp *PPv2) computeAimValue() float64 {
	rawAim := pp.aimStrain

	if pp.diff.Mods.Active(difficulty.TouchDevice) {
		rawAim = math.Pow(rawAim, 0.8)
	}

	aimValue := ppBase(rawAim)

	// Longer maps are worth more
	lengthBonus := 0.95 + 0.4*math.Min(1.0, float64(pp.totalHits)/2000.0)
	if pp.totalHits > 2000 {
		lengthBonus += math.Log10(float64(pp.totalHits)/2000.0) * 0.5
	}

	aimValue *= lengthBonus

	if pp.diff.Mods.Active(difficulty.Relax) {
		if pp.countMiss > 0 {
			aimValue *= 0.95 * math.Pow(1-math.Pow(float64(pp.countMiss)/float64(pp.totalHits), 0.775), float64(pp.countMiss))
		}
	} else {
		// Penalize misses by assessing # of misses relative to the total # of objects. Default a 3% reduction for any # of misses.
		if pp.countMiss > 0 {
			aimValue *= 0.97 * math.Pow(1-math.Pow(float64(pp.countMiss)/float64(pp.totalHits), 0.775), float64(pp.countMiss))
		}
	}

	// Combo scaling
	if pp.maxCombo > 0 {
		aimValue *= math.Min(math.Pow(float64(pp.scoreMaxCombo), 0.8)/math.Pow(float64(pp.maxCombo), 0.8), 1.0)
	}

	approachRateFactor := 0.0

	if pp.diff.Mods.Active(difficulty.Relax) {
		if pp.diff.ARReal > 10.7 {
			approachRateFactor += 0.4 * (pp.diff.ARReal - 10.7)
		} else if pp.diff.ARReal < 8.0 {
			approachRateFactor += 0.1 * (8.0 - pp.diff.ARReal)
		}
	} else { 
		if pp.diff.ARReal > 10.33 {
			approachRateFactor += 0.4 * (pp.diff.ARReal - 10.33)
		} else if pp.diff.ARReal < 8.0 {
			approachRateFactor += 0.1 * (8.0 - pp.diff.ARReal)
		}
	}

	aimValue *= 1.0 + math.Min(approachRateFactor, approachRateFactor*(float64(pp.totalHits)/1000.0))

	// We want to give more reward for lower AR when it comes to aim and HD. This nerfs high AR and buffs lower AR.
	if pp.diff.Mods.Active(difficulty.Hidden) {
		if pp.diff.Mods.Active(difficulty.Relax) {
			aimValue *= 1.0 + 0.05 * (11.0 - pp.diff.ARReal)
		} else {
			aimValue *= 1.0 + 0.04*(12.0-pp.diff.ARReal)
		}
	}

	if pp.diff.Mods.Active(difficulty.Flashlight) {
		flBonus := 1.0 + 0.35*math.Min(1.0, float64(pp.totalHits)/200.0)
		if pp.totalHits > 200 {
			flBonus += 0.3 * math.Min(1, (float64(pp.totalHits)-200.0)/300.0)
		}

		if pp.totalHits > 500 {
			flBonus += (float64(pp.totalHits) - 500.0) / 1200.0
		}

		aimValue *= flBonus
	}

	// Scale the aim value with accuracy _slightly_
	if pp.diff.Mods.Active(difficulty.Relax) {
		if (pp.diff.ODReal >= 10.6) {
			if (pp.accuracy >= 0.98) {
				aimValue *= 0.5 + pp.accuracy / 2.0
			} else if (pp.accuracy >= 0.97) {
				aimValue *= 0.47 + pp.accuracy / 2.0
			} else if (pp.accuracy >= 0.96)  {
				aimValue *= 0.45 + pp.accuracy / 2.0
			} else {
				aimValue *= 0.4 + pp.accuracy / 2.0
			}
		} else {
			if (pp.accuracy >= 0.97) {
				aimValue *= 0.4 + pp.accuracy / 2.0
			} else {
				aimValue *= 0.3 + pp.accuracy / 2.0
			}
		}
	} else {
		aimValue *= 0.5 + pp.accuracy/2.0
	}

	// It is important to also consider accuracy difficulty when doing that
	aimValue *= 0.98 + math.Pow(pp.diff.ODReal, 2)/2500

	return aimValue
}

func (pp *PPv2) computeSpeedValue() float64 {
	speedValue := ppBase(pp.speedStrain)

	// Longer maps are worth more
	lengthBonus := 0.95 + 0.4*math.Min(1.0, float64(pp.totalHits)/2000.0)
	if pp.totalHits > 2000 {
		lengthBonus += math.Log10(float64(pp.totalHits)/2000.0) * 0.5
	}

	speedValue *= lengthBonus

	speedValue *= 0.97 * math.Pow(1-math.Pow(float64(pp.countMiss)/float64(pp.totalHits), 0.775), math.Pow(float64(pp.countMiss), 0.875))

	// Combo scaling
	if pp.maxCombo > 0 {
		speedValue *= math.Min(math.Pow(float64(pp.scoreMaxCombo), 0.8)/math.Pow(float64(pp.maxCombo), 0.8), 1.0)
	}

	approachRateFactor := 0.0

	if pp.diff.Mods.Active(difficulty.Relax) {
		if pp.diff.ARReal > 10.7 {
			approachRateFactor += 0.4 * (pp.diff.ARReal - 10.7)
		} else if pp.diff.ARReal < 8.0 {
			approachRateFactor += 0.1 * (8.0 - pp.diff.ARReal)
		}
	} else { 
		if pp.diff.ARReal > 10.33 {
			approachRateFactor += 0.4 * (pp.diff.ARReal - 10.33)
		} else if pp.diff.ARReal < 8.0 {
			approachRateFactor += 0.1 * (8.0 - pp.diff.ARReal)
		}
	}

	if pp.diff.ARReal > 10.33 {
		speedValue *= 1.0 + math.Min(approachRateFactor, approachRateFactor*(float64(pp.totalHits)/1000.0))
	}

	if pp.diff.Mods.Active(difficulty.Hidden) {
		if pp.diff.Mods.Active(difficulty.Relax) {
			speedValue *= 1.0 + 0.05 * (11.0 - pp.diff.ARReal)
		} else {
			speedValue *= 1.0 + 0.04*(12.0-pp.diff.ARReal)
		}
	}

	// Scale the speed value with accuracy and OD
	speedValue *= (0.95 + math.Pow(pp.diff.ODReal, 2)/750) * math.Pow(pp.accuracy, (14.5-math.Max(pp.diff.ODReal, 8))/2)

	mehMult := 0.0
	if float64(pp.countMeh) >= float64(pp.totalHits)/500 {
		mehMult = float64(pp.countMeh) - float64(pp.totalHits)/500.0
	}

	speedValue *= math.Pow(0.98, mehMult)

	return speedValue
}

func (pp *PPv2) computeAccuracyValue() float64 {
	// This percentage only considers HitCircles of any value - in this part of the calculation we focus on hitting the timing hit window
	betterAccuracyPercentage := 0.0

	if pp.amountHitObjectsWithAccuracy > 0 {
		betterAccuracyPercentage = float64((pp.countGreat-(pp.totalHits-pp.amountHitObjectsWithAccuracy))*6+pp.countOk*2+pp.countMeh) / (float64(pp.amountHitObjectsWithAccuracy) * 6)
	}

	// It is possible to reach a negative accuracy with this formula. Cap it at zero - zero points
	if betterAccuracyPercentage < 0 {
		betterAccuracyPercentage = 0
	}

	// Lots of arbitrary values from testing.
	// Considering to use derivation from perfect accuracy in a probabilistic manner - assume normal distribution
	accuracyValue := math.Pow(1.52163, pp.diff.ODReal) * math.Pow(betterAccuracyPercentage, 24) * 2.83

	// Bonus for many hitcircles - it's harder to keep good accuracy up for longer
	accuracyValue *= math.Min(1.15, math.Pow(float64(pp.amountHitObjectsWithAccuracy)/1000.0, 0.3))

	if pp.diff.Mods.Active(difficulty.Hidden) {
		accuracyValue *= 1.08
	}

	if pp.diff.Mods.Active(difficulty.Flashlight) {
		accuracyValue *= 1.02
	}

	return accuracyValue
}
