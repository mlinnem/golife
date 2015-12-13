package lib

import (
	"math"
	"math/rand"
)

//---UKEEEP-------------
const CELL_LIFESPAN = 300

const BASE_CELL_UPKEEP = 1.0
const CANOPY_UPKEEP = 3.0
const LEGS_UPKEEP = .25
const HEIGHT_UPKEEP = .25

//---ACTION_COSTS-------
const MOVE_COST = 5
const THINKING_COST = 3.0
const REPRODUCE_COST = 30
const GROWCANOPY_COST = 120
const GROWLEGS_COST = 45
const GROWHEIGHT_COST = 150

//TODO: Make it easy to add a field and have it appear in all the right places Re: copying and whatnot
type Cell struct {
	Energy                             float64
	X                                  int
	Y                                  int
	Height                             int
	ID                                 int
	TimeLeftToWait                     int
	ClockRate                          int
	Canopy                             bool
	Legs                               bool
	GrowCanopyAt                       float64
	GrowLegsAt                         float64
	GrowHeightAt                       float64
	EnergySpentOnReproducing           float64
	NextMomentSelf                     *Cell
	Age                                int
	SpeciesID                          int
	PercentChanceWait                  int     //out of 100
	MoveChance                         float64 //out of 100
	_secretID                          int
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
	if cell.Energy > 0 && cell.NextMomentSelf != nil {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f -%6.1f\n", cell.NextMomentSelf.Energy, cell.NextMomentSelf.Energy-amt, amt)
		cell.NextMomentSelf.Energy -= amt
	}
}

func (cell *Cell) IncreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	if cell.Energy > 0 && cell.NextMomentSelf != nil {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f, +%6.1f\n", cell.NextMomentSelf.Energy, cell.NextMomentSelf.Energy+amt, amt)

		cell.NextMomentSelf.Energy = cell.NextMomentSelf.Energy + amt
	}
}

//TODO: Why dpes height trigger flip out on species, and why is height so adaptive even without canapies.
func (cell *Cell) Maintain() {
	var totalUpkeep = BASE_CELL_UPKEEP
	if cell.Legs {
		totalUpkeep += LEGS_UPKEEP
	}
	if cell.Canopy {
		totalUpkeep += CANOPY_UPKEEP
	}
	if cell.Height == 1 {
		totalUpkeep += HEIGHT_UPKEEP
	}
	if cell != nil {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d GROW STATUS: canopy %t, height %d, legs %t\n", cell.ID, cell.Canopy, cell.Height, cell.Legs)
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: maintain of %6.1f at age %d\n", cell.ID, totalUpkeep, cell.Age)
	}
	cell.DecreaseEnergy((totalUpkeep * float64(cell.Age)) / CELL_LIFESPAN)
	//cell.DecreaseEnergy(40)
	//	cell.Energy -= totalUpkeep * float64(cell.Age) / CELL_LIFESPAN
	cell.increaseAge(1)
}

func (cell *Cell) increaseAge(amt int) {
	if !cell.isDead() {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Age %d -> %d, +1\n", cell.ID, cell.Age, cell.Age+1)
		cell.NextMomentSelf.Age = cell.NextMomentSelf.Age + amt
	}
}

func (cell *Cell) IncreaseWaitTime(amt int) {
	if !cell.isDead() {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Wait time %d -> %d, +%d\n", cell.ID, cell.TimeLeftToWait, cell.TimeLeftToWait+amt, amt)
		cell.NextMomentSelf.TimeLeftToWait += amt
	}
}

func (cell *Cell) GrowHeight() {
	if cell.isDead() {
		return
	} else if !cell.IsReadyToGrowHeight() {
		cell.DecreaseEnergy(GROWHEIGHT_COST)
		return
	} else {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing height\n", cell.ID)
		cell.NextMomentSelf.Height = 1
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
		cell.NextMomentSelf.Legs = true
		cell.DecreaseEnergy(GROWLEGS_COST)
	}
}

func (cell *Cell) MoveRandom() bool {
	if !cell.WantsToAndCanMove() {
		return false
	}

	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y

		if !NextMoment.IsOccupied(xTry, yTry) {
			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d -> %d, %d\n", cell.ID, cell.X, cell.Y, xTry, yTry)
			NextMoment.CellsSpatialIndex[yTry][xTry] = cell
			NextMoment.CellsSpatialIndex[cell.Y][cell.X] = nil
			cell.X = xTry
			cell.Y = yTry
			cell.DecreaseEnergy(MOVE_COST)

			return true
		}
	}
	return false
}

func (cell *Cell) Wait() {
	if !cell.isDead() {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Waiting %d -> %d\n", cell.ID, cell.NextMomentSelf.TimeLeftToWait, cell.ClockRate*ACTUAL_WAIT_MULTIPLIER)
		cell.NextMomentSelf.TimeLeftToWait = ACTUAL_WAIT_MULTIPLIER * cell.ClockRate
	}
}

func (cell *Cell) ShouldWait() bool {
	return rand.Intn(100) < cell.PercentChanceWait
}

func (cell *Cell) isDead() bool {
	return cell.Energy <= 0 || cell.NextMomentSelf == nil
}

func (cell *Cell) IsTimeToReproduce() bool {
	var isThereASpotToReproduce = false
	for relativeY := -1; relativeY < 2; relativeY++ {
		for relativeX := -1; relativeX < 2; relativeX++ {
			var xTry = cell.X + relativeX
			var yTry = cell.Y + relativeY
			if !NextMoment.IsOccupied(xTry, yTry) {
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
	return cell.Canopy == false && cell.Energy > cell.GrowCanopyAt
}

func (cell *Cell) IsReadyToGrowHeight() bool {
	return cell.Height == 0 && cell.Energy > cell.GrowHeightAt
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
			if !NextMoment.IsOccupied(xTry, yTry) {
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
	return !cell.isDead() && cell.Legs == false && cell.Energy > cell.GrowLegsAt
}

func (cell *Cell) CountDown_TimeLeftToWait() {
	if !cell.isDead() {
		//TODO: This may not be necessary to Max
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Waiting... (%d left)\n", cell.ID, cell.NextMomentSelf.TimeLeftToWait-1)
		cell.NextMomentSelf.TimeLeftToWait = int(math.Max(0.0, float64(cell.NextMomentSelf.TimeLeftToWait-1)))
	}
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
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing Canopy\n", cell.ID)
		cell.NextMomentSelf.Canopy = true
		//TODO: Reinstate this log
		//Log(LOGTYPE_CELLEFFECT, "Cell %d grew canopy\n", cell.ID)
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
