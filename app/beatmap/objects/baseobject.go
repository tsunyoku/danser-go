package objects

import (
	"github.com/tsunyoku/danser/app/audio"
	"github.com/tsunyoku/danser/framework/graphics/batch"
	color2 "github.com/tsunyoku/danser/framework/math/color"
	"github.com/tsunyoku/danser/framework/math/vector"
	"strconv"
	"strings"
)

type Renderable interface {
	Draw(time float64, color color2.Color, batch *batch.QuadBatch) bool
	DrawApproach(time float64, color color2.Color, batch *batch.QuadBatch)
}

func commonParse(data []string, extraIndex int) *HitObject {
	x, _ := strconv.ParseFloat(data[0], 32)
	y, _ := strconv.ParseFloat(data[1], 32)
	time, _ := strconv.ParseInt(data[2], 10, 64)
	objType, _ := strconv.ParseInt(data[3], 10, 64)

	startPos := vector.NewVec2f(float32(x), float32(y))

	hitObject := &HitObject{
		StartPosRaw: startPos,
		EndPosRaw:   startPos,
		StartTime:   float64(time),
		EndTime:     float64(time),
		HitObjectID: -1,
		NewCombo:    (objType & 4) == 4,
		ColorOffset: (objType >> 4) & 7,
	}

	hitObject.BasicHitSound = parseExtras(data, extraIndex)

	return hitObject
}

func parseExtras(data []string, extraIndex int) (info audio.HitSoundInfo) {
	if extraIndex < len(data) && len(data[extraIndex]) > 0 {
		extras := strings.Split(data[extraIndex], ":")

		info.SampleSet, _ = strconv.Atoi(extras[0])
		info.AdditionSet, _ = strconv.Atoi(extras[1])

		if len(extras) > 2 {
			info.CustomIndex, _ = strconv.Atoi(extras[2])
		}

		if len(extras) > 3 {
			volume, _ := strconv.Atoi(extras[3])
			info.CustomVolume = float64(volume) / 100.0
		}
	}

	return
}
