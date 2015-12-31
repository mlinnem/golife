package lib

import (
	"fmt"
	"math"
	"sort"
)

//UI VARIABLES
var LOGTYPES_ENABLED = []int{LOGTYPE_CELLEFFECT, LOGTYPE_PRINTGRID_GRID, LOGTYPE_PRINTGRID_SUMMARY, LOGTYPE_SPECIESREPORT, LOGTYPE_FINALSTATS, LOGTYPE_ERROR}

var DEBUG = true

var CELLEFFECT_ONLY_IF_TRACED = true

var TRACEDCELL_AGECAP = 25
var TRACEDCELL_AGERANGE_EXPANSIONS = 3 //increase range by cap this many times before giving up

const (
	LOGTYPE_ERROR                  = iota
	LOGTYPE_MAINLOOPSINGLE         = iota
	LOGTYPE_MAINLOOPSINGLE_PRIMARY = iota
	LOGTYPE_FINALSTATS             = iota
	LOGTYPE_HIGHFREQUENCY          = iota
	LOGTYPE_DEBUGCONCURRENCY       = iota
	LOGTYPE_PRINTGRIDCELLS         = iota
	LOGTYPE_PRINTGRID_GRID         = iota
	LOGTYPE_PRINTGRID_BIGGRID      = iota
	LOGTYPE_PRINTGRID_SUMMARY      = iota
	LOGTYPE_OTHER                  = iota
	LOGTYPE_FAILURES               = iota
	LOGTYPE_SPECIALEVENT           = iota
	LOGTYPE_SPECIESREPORT          = iota
	LOGTYPE_CELLEFFECT             = iota
)

const PRINTGRID_EVERY_N_TURNS = 50

const DEFAULT_PRINTGRID_DEPTH = 0

const SPECIES_DIVERGENCE_THRESHOLD = 1.5

const BIGGRID_INCREMENT = 3

const NUM_TOP_SPECIES_TO_PRINT = 5

//TODO: Better place for this?
var SpeciesCounter = 0

func Log(logType int, message string, params ...interface{}) {
	if containsInt(LOGTYPES_ENABLED, logType) {
		fmt.Printf(message, params...)
	}
}

func LogIfTraced(cell *Cell, logType int, message string, params ...interface{}) bool {
	if DEBUG && ((cell != nil && CELLEFFECT_ONLY_IF_TRACED == false) || (TracedCell != nil && cell.ID == TracedCell.ID)) {
		Log(logType, "TRACED: "+message, params...)
		return true
	}
	return false
}

//should never be called if this goes through m4 macro system
func  {
	//This is used by macro processor m4 in makefile. It removes the line that calls this in some situations
	return
}

func containsInt(ints []int, theInt int) bool {
	for _, v := range ints {
		if v == theInt {
			return true
		}
	}
	return false
}

func HasSignificantGeneticDivergence(cell *Cell) bool {
	//TODO: took canopy out because something is jacked up about it. Need to debug and put back in
	//var GrowCanopyAtDiff = 0.0
	//if cell.GrowCanopyAt != 0.0 {
	//	GrowCanopyAtDiff = math.Abs(cell.X_OriginalGrowCanopyAt - cell.GrowCanopyAt)
	//}
	//TODO: Delete this crap

	var ClockRateDiff = math.Abs(float64(cell.X_originalClockRate)-float64(cell.ClockRate)) / 200
	var EnergyReproduceThresholdDiff = math.Abs(cell.X_originalEnergyReproduceThreshold-cell.EnergyReproduceThreshold) / 3000
	var EnergySpentOnReproducingDiff = math.Abs(cell.X_originalEnergySpentOnReproducing-cell.EnergySpentOnReproducing) / 3000
	var GrowCanopyAtDiff = math.Abs(float64(cell.X_originalGrowCanopyAt)-float64(cell.GrowCanopyAt)) / 3000
	var GrowChloroplastsAtDiff = math.Abs(float64(cell.X_originalGrowChloroplastsAt)-float64(cell.GrowChloroplastsAt)) / 3000
	var GrowDigestiveSystemAtDiff = math.Abs(float64(cell.X_originalGrowDigestiveSystemAt)-float64(cell.GrowDigestiveSystemAt)) / 3000
	var GrowHeightAtDiff = math.Abs(float64(cell.X_originalGrowHeightAt)-float64(cell.GrowHeightAt)) / 3000
	var GrowLegsAtDiff = math.Abs(float64(cell.X_originalGrowLegsAt)-float64(cell.GrowLegsAt)) / 3000
	var MoveChanceDiff = math.Abs(float64(cell.X_originalMoveChance)-float64(cell.MoveChance)) / 200 //reduced importance
	var PercentChanceWaitDiff = math.Abs(float64(cell.X_originalPercentChanceWait)-float64(cell.PercentChanceWait)) / 100
	var totalDiff = GrowDigestiveSystemAtDiff + GrowChloroplastsAtDiff + MoveChanceDiff + GrowLegsAtDiff + GrowHeightAtDiff + GrowCanopyAtDiff + ClockRateDiff + EnergyReproduceThresholdDiff + EnergySpentOnReproducingDiff + PercentChanceWaitDiff
	//fmt.Printf("totalDiff: %f\n", totalDiff)
	//fmt.Printf("\tClock rate diff: %f\n", ClockRateDiff)
	//fmt.Printf("\tenergy reproducing threshold diff: %f\n", EnergyReproduceThresholdDiff)
	//fmt.Printf("\tenergy spent reprod diff: %f\n", EnergySpentOnReproducingDiff)
	//fmt.Printf("\tgrew canopy at diff: %f\n", GrowCanopyAtDiff)
	//fmt.Printf("\tGrowHeightAtDiff: %f\n", GrowHeightAtDiff)
	//fmt.Printf("\tGrowLegsAtDiff: %f\n", GrowLegsAtDiff)
	//fmt.Printf("\tMoveChanceDiff: %f\n", MoveChanceDiff)
	//fmt.Printf("\tPercentChanceWaitDiff: %f\n", GrowLegsAtDiff)
	//fmt.Printf("\tTotalDiff: %f\n", totalDiff)
	//}
	return totalDiff > SPECIES_DIVERGENCE_THRESHOLD
}

func printCellAt(x, y, z int) {
	var cell, _ = WS.GetAnyCellAt(x, y, z)
	printCell(cell)
}

func printCell(cell *Cell) {
	if cell != nil {
		var colorStart = cell.SpeciesColor.StartSequence
		var colorEnd = cell.SpeciesColor.EndSequence
		if TracedCell != nil && cell.ID == TracedCell.ID && CELLEFFECT_ONLY_IF_TRACED {
			//TODO: Might be nice to make this a specific loud color at some point
			Log(LOGTYPE_PRINTGRID_GRID, "!")
			Log(LOGTYPE_PRINTGRID_BIGGRID, "!")
		} else if cell.IsAnimal() {
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"@"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"@"+colorEnd)
		} else if cell.Legs && cell.MoveChance > 0.0 {
			//curious moving trees
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"?"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"?"+colorEnd)
		} else if cell.Canopy == true && cell.Height >= 1 {
			//fmt.Println("BLOLOLOLOLO")
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"X"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"X"+colorEnd)
		} else if cell.Canopy == true {
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"x"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"x"+colorEnd)
		} else if cell.Height >= 1 {
			//fmt.Println("GLOOOOPOLOLOLOLO")
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+"O"+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"O"+colorEnd)
		} else {
			Log(LOGTYPE_PRINTGRID_GRID, colorStart+" "+colorEnd)
			Log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+" "+colorEnd)
		}
	} else {
		Log(LOGTYPE_PRINTGRID_GRID, " ")
	}
}

func PrintGrid(WS *WorldState, z int) {
	if containsInt(LOGTYPES_ENABLED, LOGTYPE_PRINTGRID_GRID) {
		Log(LOGTYPE_PRINTGRID_GRID, "\n")
		for yi := range WS.SpatialIndexSurfaceCover[z] {
			for xi := range WS.SpatialIndexSurfaceCover[z][yi] {
				//var cell = WS.SpatialIndexSurfaceCover[z][yi][xi]
				printCellAt(xi, yi, z)
			}
			Log(LOGTYPE_PRINTGRID_GRID, "\n")
		}
	} else if containsInt(LOGTYPES_ENABLED, LOGTYPE_PRINTGRID_BIGGRID) {
		Log(LOGTYPE_PRINTGRID_BIGGRID, "\n")
		for yi := 0; yi < GRID_HEIGHT; yi += BIGGRID_INCREMENT {
			for xi := 0; xi < GRID_WIDTH; xi += BIGGRID_INCREMENT {
				//	var cell = WS.SpatialIndexSurfaceCover[z][yi][xi]
				printCellAt(xi, yi, z)
			}
			Log(LOGTYPE_PRINTGRID_BIGGRID, "\n")
		}
	}
	Log(LOGTYPE_PRINTGRID_GRID, "\n")

	//TODO: Unnecessary capitalization here
	var energyReproduceThresholdTotal = 0.0
	var EnergySpentOnReproducingTotal = 0.0
	var canopyTotal = 0
	var GrowCanopyAtTotal = 0.0
	var GrowHeightAtTotal = 0.0
	var PercentChanceWaitTotal = 0
	var ClockRateTotal = 0
	var MoveChanceTotal = 0.0
	var legsTotal = 0
	var energyTotal = 0.0
	var ageTotal = 0
	var chloroplastsTotal = 0
	var height1Total = 0
	var digestiveSystemTotal = 0
	var growDigestiveSystemAtTotal = 0.0

	for ci := range WS.Cells {
		var cell = WS.Cells[ci]

		ageTotal += cell.Age
		energyTotal += cell.Energy
		GrowHeightAtTotal += cell.GrowHeightAt
		GrowCanopyAtTotal += cell.GrowCanopyAt
		energyReproduceThresholdTotal += cell.EnergyReproduceThreshold
		EnergySpentOnReproducingTotal += cell.EnergySpentOnReproducing
		PercentChanceWaitTotal += cell.PercentChanceWait
		ClockRateTotal += cell.ClockRate
		growDigestiveSystemAtTotal += cell.GrowDigestiveSystemAt

		if WS.Cells[ci].Canopy == true {
			canopyTotal++
		}
		if cell.Legs == true {
			legsTotal++
			MoveChanceTotal += cell.MoveChance
		}

		if cell.Chloroplasts == true {
			chloroplastsTotal++
		}

		if cell.Height == 1 {
			height1Total++
		}

		if cell.DigestiveSystem == true {
			digestiveSystemTotal++
		}

		Log(LOGTYPE_PRINTGRIDCELLS, "(Cell) %d: %d,%d with %f, age %d, reprod at %f, grew canopy at %f, reproduces with %f\n", cell.ID, WS.Cells[ci].X, WS.Cells[ci].Y, WS.Cells[ci].Energy, WS.Cells[ci].Age, WS.Cells[ci].EnergyReproduceThreshold, WS.Cells[ci].GrowCanopyAt, cell.EnergySpentOnReproducing)
	}

	Log(LOGTYPE_PRINTGRID_SUMMARY, "-----SUMMARY STATE-----\n")
	Log(LOGTYPE_PRINTGRID_SUMMARY, "WS %d...\n", WS.WSNum)

	Log(LOGTYPE_PRINTGRID_SUMMARY, "%d cells total\n\n", len(WS.Cells))
	Log(LOGTYPE_MAINLOOPSINGLE, "Cell Count: %d\n", len(WS.Cells))
	var energyReproduceThresholdAvg = energyReproduceThresholdTotal / float64(len(WS.Cells))
	var GrowCanopyAtAvg = GrowCanopyAtTotal / float64(len(WS.Cells))
	var GrowHeightAtAvg = GrowHeightAtTotal / float64(len(WS.Cells))
	var EnergySpentOnReproducingAvg = EnergySpentOnReproducingTotal / float64(len(WS.Cells))
	var PercentChanceWaitAvg = float64(PercentChanceWaitTotal) / float64(len(WS.Cells))
	var ClockRateAvg = float64(ClockRateTotal) / float64(len(WS.Cells))
	var percentMoveAvg = float64(MoveChanceTotal) / float64(legsTotal)
	var energyAvg = float64(energyTotal) / float64(len(WS.Cells))
	var ageAvg = float64(ageTotal) / float64(len(WS.Cells))
	var chloroplastsPercent = 100.0 * float64(chloroplastsTotal) / float64(len(WS.Cells))
	var growDigestiveSystemAtAvg = float64(growDigestiveSystemAtTotal) / float64(len(WS.Cells))

	Log(LOGTYPE_PRINTGRID_SUMMARY, "Average age: %6.1f\n", ageAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Average energy: %6.1f\n", energyAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Wait Average: %6.1f\n", PercentChanceWaitAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Average Clock Rate: %6.1f\n", ClockRateAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Reproduce Threshold Average: %6.1f\n", energyReproduceThresholdAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Spent on Reproducing Average: %6.1f\n", EnergySpentOnReproducingAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Chloroplasts Percent: %6.1f\n", chloroplastsPercent)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Canopy Percent: %6.1f\n", 100.0*float64(canopyTotal)/float64(len(WS.Cells)))
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Grow Canopy At Average: %6.1f\n", GrowCanopyAtAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Height1 Percent: %6.1f\n", 100.0*float64(height1Total)/float64(len(WS.Cells)))
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Grow Height At Average: %6.1f\n", GrowHeightAtAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Legs Percent: %6.1f\n", 100.0*float64(legsTotal)/float64(len(WS.Cells)))
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Move (with legs) Average: %6.1f\n", percentMoveAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Digestive System Percent: %6.1f\n", 100.0*float64(digestiveSystemTotal)/float64(len(WS.Cells)))
	Log(LOGTYPE_PRINTGRID_SUMMARY, "Grow Digestive System At Average: %6.1f\n", growDigestiveSystemAtAvg)
	Log(LOGTYPE_PRINTGRID_SUMMARY, "New species so far: %d\n", SpeciesCounter)

	Log(LOGTYPE_PRINTGRID_SUMMARY, "\n")
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func PrintSpeciesReport(WS *WorldState, topXSpecies int) {
	var SpeciesIDToCount = make(map[string]int)
	var SpeciesIDToSpecimen = make(map[string]*Cell)
	for ci := range WS.Cells {
		var cell = WS.Cells[ci]
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

		//TODO: Need to report species number right. Dot for now
		LogIfTraced(specimen, LOGTYPE_CELLEFFECT, "%d\n", count)

		//TODO: Re-enable this once bug is found in this line
		//	Log(LOGTYPE_SPECIESREPORT, "Species %s\t "+specimen.SpeciesColor.StartSequence+"x"+specimen.SpeciesColor.EndSequence+"\t"+"\t"+"Count: %d\t reprod threshold: %6.1f\t reprod energy: %6.1f\t gcanopy thresh: %6.1f\t wait percent: %d\t clock rate %d\t gleg thresh: %6.1f\t, moveChance: %6.1f\t, growHeightAt %6.1f growChloroplastsAt %6.1f\t growDigestiveSystemAt %6.1f\n", ".", count, specimen.X_originalEnergyReproduceThreshold, specimen.X_originalEnergySpentOnReproducing, specimen.X_originalGrowCanopyAt, specimen.X_originalPercentChanceWait, specimen.X_originalClockRate, specimen.X_originalGrowLegsAt, specimen.X_originalMoveChance, specimen.X_originalGrowHeightAt, specimen.X_originalGrowChloroplastsAt, specimen.X_originalGrowDigestiveSystemAt)
	}
	Log(LOGTYPE_SPECIESREPORT, "\n")
}

func PrintCurrentGenesOfCell(specimen *Cell) {
	Log(LOGTYPE_SPECIESREPORT, GetStringOfCurrentGenesOfCell(specimen))
}

func GetStringOfCurrentGenesOfCell(specimen *Cell) string {
	return fmt.Sprintf("cell %d GENES: Species %s\t "+specimen.SpeciesColor.StartSequence+"x"+specimen.SpeciesColor.EndSequence+"\t"+"\t"+"Count: %d\t reprod threshold: %6.1f\t reprod energy: %6.1f\t gcanopy thresh: %6.1f\t wait percent: %d\t clock rate %d\t gleg thresh: %6.1f\t, moveChance: %6.1f\t, growHeightAt %6.1f growChloroplastsAt %6.1f\t growDigestiveSystemAt %6.1f\n",
		specimen.ID, ".", 1, specimen.EnergyReproduceThreshold, specimen.EnergySpentOnReproducing, specimen.GrowCanopyAt, specimen.PercentChanceWait, specimen.ClockRate, specimen.GrowLegsAt, specimen.MoveChance, specimen.GrowHeightAt, specimen.GrowChloroplastsAt, specimen.GrowDigestiveSystemAt)
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
