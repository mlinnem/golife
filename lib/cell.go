package lib

import (
	"math"
	"math/rand"
)

//---UKEEEP-------------
const CELL_LIFESPAN = 300

const BASIC_BRAIN_UPKEEP = .25
const CHLOROPLAST_UPKEEP = .75
const CANOPY_UPKEEP = 3.5
const LEGS_UPKEEP = .25
const HEIGHT_UPKEEP = .25

//---ACTION_COSTS-------
const MOVE_COST = 5
const THINKING_COST = 10.0
const REPRODUCE_COST = 30

const GROWCHLOROPLASTS_COST = 15
const GROWCANOPY_COST = 60
const GROWLEGS_COST = 45
const GROWHEIGHT_COST = 200

//----OTHER-------------
const ACTUAL_WAIT_MULTIPLIER = 3

//TODO: Make it easy to add a field and have it appear in all the right places Re: copying and whatnot

//TODO: Might need to do audit for proper handling of 'Z'
type Cell struct {
	Energy                             float64
	X                                  int
	Y                                  int
	Z                                  int
	ID                                 int
	TimeLeftToWait                     int
	ClockRate                          int
	Canopy                             bool
	Legs                               bool
	Chloroplasts                       bool
	Height                             int
	GrowChloroplastsAt                 float64
	GrowCanopyAt                       float64
	GrowLegsAt                         float64
	GrowHeightAt                       float64
	EnergySpentOnReproducing           float64
	Age                                int
	SpeciesID                          int
	PercentChanceWait                  int     //out of 100
	MoveChance                         float64 //out of 100
	_secretID                          int
	X_originalGrowChloroplastsAt       float64
	X_originalGrowHeightAt             float64
	X_originalGrowCanopyAt             float64
	X_originalPercentChanceWait        int
	X_originalEnergySpentOnReproducing float64
	X_originalEnergyReproduceThreshold float64
	X_originalClockRate                int
	X_originalGrowLegsAt               float64
	X_originalMoveChance               float64
	EnergyReproduceThreshold           float64
	SpeciesColor                       *TextColorBookend
}

type TextColorBookend struct {
	StartSequence string
	EndSequence   string
}

var TracedCell *Cell

func (cell *Cell) DecreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	//if cell.Energy > 0 && cell.WSSelf != nil {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f -%6.1f\n", cell.Energy, cell.Energy-amt, amt)
	cell.Energy -= amt
	//}
}

func (cell *Cell) IncreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	//if cell.Energy > 0 && cell.WSSelf != nil {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f, +%6.1f\n", cell.Energy, cell.Energy+amt, amt)

	cell.Energy = cell.Energy + amt
	//}
}

//TODO: Why dpes height trigger flip out on species, and why is height so adaptive even without canapies.
func (cell *Cell) Maintain() {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Starting maintain\n", cell.ID)
	var totalUpkeep = BASIC_BRAIN_UPKEEP
	if cell.Chloroplasts {
		totalUpkeep += CHLOROPLAST_UPKEEP
	}
	if cell.Legs {
		totalUpkeep += LEGS_UPKEEP
	}
	if cell.Canopy {
		totalUpkeep += CANOPY_UPKEEP
	}
	if cell.Height == 1 {
		totalUpkeep += HEIGHT_UPKEEP
	}
	//	if cell != nil {
	//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d GROW STATUS: energy %6.1f, canopy %t, height %d, legs %t, chloroplasts %t,\n", cell.ID, cell.Energy, cell.Canopy, cell.Height, cell.Legs, cell.Chloroplasts)
	//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: maintain of %6.1f at age %d\n", cell.ID, totalUpkeep, cell.Age)
	//	}
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: About to decrease energy from maintain\n", cell.ID)
	cell.DecreaseEnergy((totalUpkeep * float64(cell.Age)) / CELL_LIFESPAN)
	cell.increaseAge(1)
}

func (cell *Cell) increaseAge(amt int) {
	//if !cell.isDead() {
	//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Age %d -> %d, +1\n", cell.ID, cell.Age, cell.Age+1)
	cell.Age = cell.Age + amt
	//}
}

func (cell *Cell) IncreaseWaitTime(amt int) {
	//if !cell.isDead() {
	//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Wait time %d -> %d, +%d\n", cell.ID, cell.TimeLeftToWait, cell.TimeLeftToWait+amt, amt)
	cell.TimeLeftToWait += amt
	//}
}

func (cell *Cell) GrowHeight() {
	if cell.isDead() {
		return
	} else if !cell.IsReadyToGrowHeight() {
		cell.DecreaseEnergy(GROWHEIGHT_COST)
		return
	} else {
		//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing height\n", cell.ID)
		//TODO: This can probably be done more efficiently if it's a big deal
		WS.RemoveCellFromSpatialIndex(cell)
		cell.Height = 1
		WS.AddCellToSpatialIndex(cell)
		cell.DecreaseEnergy(GROWHEIGHT_COST)
	}
}

func (cell *Cell) GrowLegs() {
	if cell.isDead() {
		return
	} else if !cell.IsReadyToGrowLegs() {
		cell.DecreaseEnergy(GROWLEGS_COST)
		return
	} else {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing legs\n", cell.ID)
		cell.Legs = true
		cell.DecreaseEnergy(GROWLEGS_COST)
	}
}

func (cell *Cell) GrowChloroplasts() {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Trying to grow chloroplasts\n", cell.ID)

	if cell.isDead() {
		return
	} else if !cell.IsReadyToGrowChloroplasts() {
		cell.DecreaseEnergy(GROWCHLOROPLASTS_COST)
		return
	} else {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing chloroplasts\n", cell.ID)
		cell.Chloroplasts = true
		cell.DecreaseEnergy(GROWCHLOROPLASTS_COST)
	}
}

//Only moves in Z plane
func (cell *Cell) MoveRandom() bool {
	if !cell.WantsToAndCanMove() {
		return false
	}

	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y

		//TODO: This move logic may overwrite stuff for larger cells
		if WS.CanMoveHere(cell, xTry, yTry, cell.Z) {
			//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)
			//	Log(LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

			WS.RemoveCellFromSpatialIndex(cell)
			cell.X = xTry
			cell.Y = yTry
			WS.AddCellToSpatialIndex(cell)
			cell.DecreaseEnergy(MOVE_COST)

			return true
		}
	}
	return false
}

func (cell *Cell) Wait() {
	//if !cell.isDead() {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Waiting %d -> %d\n", cell.ID, cell.TimeLeftToWait, cell.ClockRate*ACTUAL_WAIT_MULTIPLIER)
	cell.TimeLeftToWait = ACTUAL_WAIT_MULTIPLIER * cell.ClockRate
	//	}
}

func (cell *Cell) ShouldWait() bool {
	return rand.Intn(100) < cell.PercentChanceWait
}

func (cell *Cell) isDead() bool {
	return cell.Energy <= 0
}

func (cell *Cell) IsTimeToReproduce() bool {
	var isThereASpotToReproduce = false
	for relativeY := -1; relativeY < 2; relativeY++ {
		for relativeX := -1; relativeX < 2; relativeX++ {
			var xTry = cell.X + relativeX
			var yTry = cell.Y + relativeY
			if !WS.IsSolidOrCovered(xTry, yTry, cell.Z) {
				isThereASpotToReproduce = true
				goto foundSpot
			}
		}
	}

foundSpot:
	if !isThereASpotToReproduce {
		return false
	}
	return cell.Energy > cell.EnergyReproduceThreshold
}

func (cell *Cell) IsReadyToGrowCanopy() bool {
	return cell.Canopy == false && cell.Chloroplasts == true && cell.Energy > cell.GrowCanopyAt
}

func (cell *Cell) IsReadyToGrowChloroplasts() bool {
	return cell.Chloroplasts == false && cell.Energy > cell.GrowChloroplastsAt
}

func (cell *Cell) IsReadyToGrowHeight() bool {
	return cell.Height == 0 && cell.Energy > cell.GrowHeightAt && !WS.IsSolidOrCovered(cell.X, cell.Y, 1)
}

func (cell *Cell) WantsToAndCanMove() bool {
	//TODO: Do we need to be doing this on each pre-check?
	if cell.isDead() || cell.Legs == false {
		return false
	}
	var isThereASpotToMove = false
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = cell.X + relativeX
			var yTry = cell.Y + relativeY
			if !WS.IsSolidOrCovered(xTry, yTry, cell.Z) {
				isThereASpotToMove = true
				goto foundSpot
			}
		}
	}

foundSpot:
	if !isThereASpotToMove {
		return false
	}
	return cell.Energy > MOVE_COST
}

func (cell *Cell) IsReadyToGrowLegs() bool {
	return !cell.isDead() && cell.Height >= 1 && cell.Legs == false && cell.Energy > cell.GrowLegsAt
}

func (cell *Cell) CountDown_TimeLeftToWait() {
	//	if !cell.isDead() {
	//TODO: This may not be necessary to Max
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Waiting... (%d left)\n", cell.ID, cell.TimeLeftToWait-1)
	cell.TimeLeftToWait = int(math.Max(0.0, float64(cell.TimeLeftToWait-1)))
	//}
}

func (cell *Cell) GrowCanopy() {
	if cell.isDead() {
		return
	} else if cell.Energy < GROWCANOPY_COST {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.DecreaseEnergy(GROWCANOPY_COST)
		return
		//TODO: Not sure why I need to check this condition twice, but it seems to prevent a nil reference thing. Or does it?
	} else if !cell.isDead() {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing Canopy\n", cell.ID)
		cell.Canopy = true
		cell.DecreaseEnergy(GROWCANOPY_COST)
	}
}

var surroundingDirections = []Direction{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

func GetSurroundingDirectionsInRandomOrder() []Direction {
	var surroundingDirectionsInRandomOrder = []Direction{}
	for _, i := range rand.Perm(len(surroundingDirections)) {
		surroundingDirectionsInRandomOrder = append(surroundingDirectionsInRandomOrder, surroundingDirections[i])
	}
	return surroundingDirectionsInRandomOrder
}
