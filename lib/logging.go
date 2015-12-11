package lib

import (
	"fmt"
	"math"
	"sort"
)

//TODO: Better place for this?
var SpeciesCounter = 0

func Log(logType int, message string, params ...interface{}) {
	if containsInt(LOGTYPES_ENABLED, logType) {
		fmt.Printf(message, params...)
	}
}

func containsInt(ints []int, theInt int) bool {
	for _, v := range ints {
		if v == theInt {
			return true
		}
	}
	return false
}

func hasSignificantGeneticDivergence(cell *Cell) bool {
	var energyReproduceThresholdDiff = math.Abs(cell.X_originalEnergyReproduceThreshold - cell.EnergyReproduceThreshold)
	//TODO: took canopy out because something is jacked up about it. Need to debug and put back in
	//var GrowCanopyAtDiff = 0.0
	//if cell.GrowCanopyAt != 0.0 {
	//	GrowCanopyAtDiff = math.Abs(cell.X_OriginalGrowCanopyAt - cell.GrowCanopyAt)
	//}

	var MoveChanceDiff = math.Abs(float64(cell.X_originalMoveChance) - float64(cell.MoveChance))
	var GrowLegsAtDiff = math.Abs(float64(cell.X_originalGrowLegsAt) - float64(cell.GrowCanopyAt))
	var GrowCanopyAtDiff = math.Abs(float64(cell.X_originalGrowCanopyAt) - float64(cell.GrowCanopyAt))
	var ClockRateDiff = math.Abs(float64(cell.X_originalClockRate) - float64(cell.ClockRate))
	var EnergySpentOnReproducingDiff = math.Abs(cell.X_originalEnergySpentOnReproducing - cell.EnergySpentOnReproducing)
	var PercentChanceWaitDiff = math.Abs(float64(cell.X_originalPercentChanceWait) - float64(cell.PercentChanceWait))
	var totalDiff = GrowLegsAtDiff + MoveChanceDiff + GrowLegsAtDiff + GrowCanopyAtDiff + ClockRateDiff + energyReproduceThresholdDiff + EnergySpentOnReproducingDiff + PercentChanceWaitDiff
	//if totalDiff > SPECIES_DIVERGENCE_THRESHOLD {
	//	Log("X_original energy threshold: %f\n", cell._X_originalEnergyReproduceThreshold)
	//	fmt.Printf("current energy threshold: %f\n", cell.EnergyReproduceThreshold)
	//	fmt.Printf("totalDiff: %f\n", totalDiff)
	//	fmt.Printf("energy spent reprod diff: %f\n", EnergySpentOnReproducingDiff)
	//	fmt.Printf("grew canopy at diff: %f\n", GrowCanopyAtDiff)
	//	fmt.Printf("energy threshold diff: %f\n", energyReproduceThresholdDiff)
	//}
	return totalDiff > SPECIES_DIVERGENCE_THRESHOLD
}

func printCell(cell *Cell) {
	if cell != nil {
		var colorStart = cell.SpeciesColor.StartSequence
		var colorEnd = cell.SpeciesColor.EndSequence
		if cell.Canopy == true {
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"X"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"X"+colorEnd)
		} else {
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"x"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"x"+colorEnd)
		}
	} else {
		Log(LOGTYPE_PRINTGRID_GRID, " ")
	}
}

func PrintGrid(moment *Moment) {
	if containsInt(LOGTYPES_ENABLED, LOGTYPE_PRINTGRID_GRID) {
		Log(LOGTYPE_PRINTGRID_GRID, "\n")
		for row := range moment.CellsSpatialIndex {
			for col := range moment.CellsSpatialIndex[row] {
				var cell = moment.CellsSpatialIndex[row][col]
				printCell(cell)
			}
			Log(LOGTYPE_PRINTGRID_GRID, "\n")
		}
	} else if containsInt(LOGTYPES_ENABLED, LOGTYPE_PRINTGRID_BIGGRID) {
		Log(LOGTYPE_PRINTGRID_BIGGRID, "\n")
		for row := 0; row < len(moment.CellsSpatialIndex); row += BIGGRID_INCREMENT {
			for col := 0; col < len(moment.CellsSpatialIndex); col += BIGGRID_INCREMENT {
				var cell = moment.CellsSpatialIndex[row][col]
				printCell(cell)
			}
			Log(LOGTYPE_PRINTGRID_BIGGRID, "\n")
		}
	}
	Log(LOGTYPE_PRINTGRID_GRID, "\n")

	var energyReproduceThresholdTotal = 0.0
	var EnergySpentOnReproducingTotal = 0.0
	var canopyTotal = 0
	var GrowCanopyAtTotal = 0.0
	var PercentChanceWaitTotal = 0
	var ClockRateTotal = 0
	var MoveChanceTotal = 0.0
	var legsTotal = 0
	for ci := range moment.Cells {
		var cell = moment.Cells[ci]

		energyReproduceThresholdTotal += cell.EnergyReproduceThreshold
		EnergySpentOnReproducingTotal += cell.EnergySpentOnReproducing
		PercentChanceWaitTotal += cell.PercentChanceWait
		ClockRateTotal += cell.ClockRate
		if moment.Cells[ci].Canopy == true {
			canopyTotal++
			GrowCanopyAtTotal += moment.Cells[ci].GrowCanopyAt
		}
		if cell.Legs == true {
			legsTotal++
			MoveChanceTotal += cell.MoveChance
		}
		Log(LOGTYPE_PRINTGRIDCELLS, "(Cell) %d: %d,%d with %f, age %d, reprod at %f, grew canopy at %f, reproduces with %f\n", cell.ID, moment.Cells[ci].X, moment.Cells[ci].Y, moment.Cells[ci].Energy, moment.Cells[ci].Age, moment.Cells[ci].EnergyReproduceThreshold, moment.Cells[ci].GrowCanopyAt, cell.EnergySpentOnReproducing)
	}

	Log(LOGTYPE_PRINTGRID_SUMMARY, "-----SUMMARY STATE-----\n")
	Log(LOGTYPE_PRINTGRID_SUMMARY, "moment %d...\n", moment.MomentNum)

	Log(LOGTYPE_PRINTGRID_SUMMARY, "%d cells total\n\n", len(moment.Cells))
	Log(LOGTYPE_MAINLOOPSINGLE, "Cell Count: %d\n", len(CurrentMoment.Cells))
	var energyReproduceThresholdAvg = energyReproduceThresholdTotal / float64(len(moment.Cells))
	var GrowCanopyAtAvg = GrowCanopyAtTotal / float64(canopyTotal)
	var EnergySpentOnReproducingAvg = EnergySpentOnReproducingTotal / float64(len(moment.Cells))
	var PercentChanceWaitAvg = float64(PercentChanceWaitTotal) / float64(len(moment.Cells))
	var ClockRateAvg = float64(ClockRateTotal) / float64(len(moment.Cells))
	var percentMoveAvg = float64(MoveChanceTotal) / float64(legsTotal)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Reproduce Threshold Average: %f\n", energyReproduceThresholdAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Spent on Reproducing Average: %f\n", EnergySpentOnReproducingAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Wait Average: %f\n", PercentChanceWaitAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Average Clock Rate: %f\n", ClockRateAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Canopy Total: %d\n", canopyTotal)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Grew Canopy At Average: %f\n", GrowCanopyAtAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Legs Total: %d\n", legsTotal)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Move (with legs) Average: %f\n", percentMoveAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "New species so far: %d\n", SpeciesCounter)

	Log(LOGTYPE_PRINTGRID_SUMMARY, "\n\n\n")
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func PrintSpeciesReport(moment *Moment, topXSpecies int) {
	var SpeciesIDToCount = make(map[string]int)
	var SpeciesIDToSpecimen = make(map[string]*Cell)
	for ci := range moment.Cells {
		var cell = moment.Cells[ci]
		var SpeciesIDString = string(cell.SpeciesID)
		var speciesCount, exists = SpeciesIDToCount[SpeciesIDString]
		if exists {
			SpeciesIDToCount[SpeciesIDString] = speciesCount + 1
		} else {
			SpeciesIDToCount[SpeciesIDString] = 1
			SpeciesIDToSpecimen[SpeciesIDString] = cell
		}
	}

	var SpeciesIDByCount = rankByCount(SpeciesIDToCount)
	var realTopXSpecies = int(math.Min(float64(topXSpecies), float64(len(SpeciesIDByCount))))
	var topXSpeciesIDByCount = SpeciesIDByCount[:realTopXSpecies]
	Log(LOGTYPE_SPECIESREPORT, "\n")
	Log(LOGTYPE_SPECIESREPORT, "-----SPECIES REPORT----\n")
	for _, pair := range topXSpeciesIDByCount {
		var SpeciesID = pair.Key
		var count = SpeciesIDToCount[SpeciesID]
		//TODO: This bugged
		var specimen *Cell = SpeciesIDToSpecimen[SpeciesID]

		Log(LOGTYPE_SPECIESREPORT, "Species %s\t "+specimen.SpeciesColor.StartSequence+"x"+specimen.SpeciesColor.EndSequence+"\t"+"\t"+"Count: %d\t reprod threshold: %6.1f\t reprod energy: %6.1f\t gcanopy thresh: %6.1f\t wait percent: %d\t clock rate %d\t gleg thresh: %6.1f\n",
			SpeciesID, count, specimen.X_originalEnergyReproduceThreshold, specimen.X_originalEnergySpentOnReproducing, specimen.X_originalGrowCanopyAt, specimen.X_originalPercentChanceWait, specimen.X_originalClockRate, specimen.X_originalGrowLegsAt)
	}
	Log(LOGTYPE_SPECIESREPORT, "\n")
}

//from http://stackoverflow.com/questions/18695346/how-to-sort-a-mapstringint-by-its-values
func rankByCount(wordFrequencies map[string]int) PairList {
	pl := make(PairList, len(wordFrequencies))
	i := 0
	for k, v := range wordFrequencies {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}
