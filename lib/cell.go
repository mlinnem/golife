package lib

import (
	"math"
	"math/rand"
)

//---UKEEEP-------------
const CELL_LIFESPAN = 500

const BASIC_BRAIN_UPKEEP = .15
const CHLOROPLAST_UPKEEP = .5
const CANOPY_UPKEEP = 2.0
const LEGS_UPKEEP = .2
const HEIGHT_UPKEEP = -0.1
const DIGESTIVESYSTEM_UPKEEP = .5

//---ACTION_COSTS-------
//--MOVE COSTS--
const BASIC_BRAIN_MOVE_COST = .0125
const CHLOROPLAST_MOVE_COST = .0125
const CANOPY_MOVE_COST = .05
const LEGS_MOVE_COST = .05
const HEIGHT_MOVE_COST = .15
const DIGESTIVESYSTEM_MOVE_COST = .0125

const EAT_COST = 1.0
const THINKING_COST = 3.0
const REPRODUCE_COST = 30

const GROWCHLOROPLASTS_COST = 25
const GROWCANOPY_COST = 100
const GROWLEGS_COST = 50
const GROWHEIGHT_COST = 225
const GROWDIGESTIVESYSTEM_COST = 25

//----OTHER-------------
const ACTUAL_WAIT_MULTIPLIER = 3
const FRACTION_EATEN_PER_EAT = .2
const EAT_MAX = 100
const CANOPY_COVER_FRACTION = .33

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
	DigestiveSystem                    bool
	Chloroplasts                       bool
	Height                             int
	GrowDigestiveSystemAt              float64
	GrowChloroplastsAt                 float64
	GrowCanopyAt                       float64
	GrowLegsAt                         float64
	GrowHeightAt                       float64
	EnergySpentOnReproducing           float64
	Age                                int
	SpeciesID                          int
	PercentChanceWait                  int     //out of 100
	MoveChance                         float64 //out of 100
	EatChance                          float64 //out of 100
	_secretID                          int
	X_originalGrowDigestiveSystemAt    float64
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
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f -%6.1f\n", cell.Energy, cell.Energy-amt, amt)
	cell.Energy -= amt
	//}
}

func (cell *Cell) IncreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	//if cell.Energy > 0 && cell.WSSelf != nil {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f, +%6.1f\n", cell.Energy, cell.Energy+amt, amt)
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\tenergy %6.1f -> %6.1f, +%6.1f\n", cell.Energy, cell.Energy+amt, amt)

	cell.Energy = cell.Energy + amt
	//}
}

//TODO: Why dpes height trigger flip out on species, and why is height so adaptive even without canapies.
func (cell *Cell) Maintain() {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Starting maintain\n", cell.ID)
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

	if cell.DigestiveSystem {
		totalUpkeep += DIGESTIVESYSTEM_UPKEEP
	}
	//	if cell != nil {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d LOCATION: %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z)
	if cell.Legs == true {
		//	var coveringCell, isCoveringCell = WS.GetCoveringCellAt(cell.X, cell.Y, cell.Z)
		//	if isCoveringCell {
		//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "\t is covered by cell %d: energy %6.1f\n", coveringCell.ID, coveringCell.Energy)
		//	}
	}
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d GROW STATUS: energy %6.1f, canopy %t, height %d, legs %t, chloroplasts %t, DigestiveSystem %t\n", cell.ID, cell.Energy, cell.Canopy, cell.Height, cell.Legs, cell.Chloroplasts, cell.DigestiveSystem)
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, GetStringOfCurrentGenesOfCell(cell))
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: maintain of %6.1f at age %d\n", cell.ID, totalUpkeep, cell.Age)
	//	}
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: About to decrease energy from maintain\n", cell.ID)
	cell.DecreaseEnergy((totalUpkeep * float64(cell.Age)) / CELL_LIFESPAN)
	//	cell.DecreaseEnergy(totalUpkeep/2 + (totalUpkeep*float64(cell.Age))/CELL_LIFESPAN)
	cell.increaseAge(1)
}

func (cell *Cell) increaseAge(amt int) {
	//if !cell.isDead() {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Age %d -> %d, +1\n", cell.ID, cell.Age, cell.Age+1)
	cell.Age = cell.Age + amt
	//}
}

func (cell *Cell) IncreaseWaitTime(amt int) {
	//if !cell.isDead() {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Wait time %d -> %d, +%d\n", cell.ID, cell.TimeLeftToWait, cell.TimeLeftToWait+amt, amt)
	cell.TimeLeftToWait += amt
	//}
}

func (cell *Cell) IsAnimal() bool {
	return cell.DigestiveSystem == true
}

func (cell *Cell) GrowDigestiveSystem() {
	if cell.isDead() {
		return
	} else if !cell.IsReadyToGrowDigestiveSystem() {
		cell.DecreaseEnergy(GROWDIGESTIVESYSTEM_COST)
		return
	} else {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing digestive system\n", cell.ID)
		cell.DigestiveSystem = true
		cell.DecreaseEnergy(GROWDIGESTIVESYSTEM_COST)
	}
}

func (cell *Cell) GrowHeight() {
	if !cell.IsReadyToGrowHeight() {
		cell.DecreaseEnergy(GROWHEIGHT_COST)
		return
	} else {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing height\n", cell.ID)
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
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing legs\n", cell.ID)
		WS.RemoveCellFromSpatialIndex(cell)
		cell.Legs = true
		WS.AddCellToSpatialIndex(cell)

		cell.DecreaseEnergy(GROWLEGS_COST)

	}
}

func (cell *Cell) GrowChloroplasts() {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Trying to grow chloroplasts\n", cell.ID)

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
func (cell *Cell) Eat() bool {
	if !cell.WantsToAndCanEat() {
		return false
	}

	var eatableCellInLocation, _ = WS.GetCoveringCellAt(cell.X, cell.Y, cell.Z)
	var energyToTake = math.Min(EAT_MAX, eatableCellInLocation.Energy*FRACTION_EATEN_PER_EAT)
	//fmt.Println("SOMEONE ATE! FOR %6.1f \n", energyToTake)
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Eating %6.1f (out of %6.1f) rom cell %d\n", cell.ID, energyToTake, eatableCellInLocation.Energy, eatableCellInLocation.ID)
	//LogIfTraced(eatableCellInLocation, LOGTYPE_CELLEFFECT, "cell %d: BEING EATEN %f from cell %d\n", eatableCellInLocation.ID, energyToTake, cell.ID)
	eatableCellInLocation.DecreaseEnergy(energyToTake)
	cell.IncreaseEnergy(energyToTake)
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: paying eat cost\n", cell.ID)
	cell.DecreaseEnergy(EAT_COST)
	return true
}

//Only moves in Z plane
func (cell *Cell) MoveRandom() bool {
	if !cell.CanMove() {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE FAILED. Tried moving %d, %d, %d but failed wantstoandcanmove\n", cell.ID, cell.X, cell.Y, cell.Z)

		return false
	}

	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y

		//TODO: This move logic may overwrite stuff for larger cells
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Checking if can move from %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

		if WS.CanMoveHere(cell, xTry, yTry, cell.Z) {
			//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

			//		var coveringCellInThisLocation, hasCoveringCell = WS.GetCoveringCellAt(xTry, yTry, cell.Z)
			//	if hasCoveringCell {
			//			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: the grid location we're moving to (%d, %d, %d) is covered by cell %d with %6.1f energy\n", cell.ID, cell.X, cell.Y, cell.Z, coveringCellInThisLocation.ID, coveringCellInThisLocation.Energy)
			//	} else {
			//			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: the grid location we're moving to (%d, %d, %d) has NO covering cell\n", cell.ID, cell.X, cell.Y, cell.Z)

			//	}

			//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

			//	Log(LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

			cell.moveHere(xTry, yTry)

			return true
		}
	}
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE FAILED. Unable to find any free spot to move near %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z)

	return false
}

//For internal use only. Still need to do checks to make sure there's no overwrite in calling functions
func (cell *Cell) moveHere(xTarget, yTarget int) {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTarget, yTarget, cell.Z)

	WS.RemoveCellFromSpatialIndex(cell)
	cell.X = xTarget
	cell.Y = yTarget
	WS.AddCellToSpatialIndex(cell)
	cell.DecreaseEnergy(cell.GetMoveCost())
}

func (cell *Cell) MoveToHighestEdibleEnergy() bool {
	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Attempting move to highest edible energy.\n", cell.ID)

	if !cell.CanMove() {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE FAILED. Tried moving %d, %d, %d but failed wantstoandcanmove\n", cell.ID, cell.X, cell.Y, cell.Z)
		return false
	}

	var foundSomewhereWithEnergy = false
	var highestEdibleEnergy = -1.0
	var highestEdibleEnergyX = -1
	var highestEdibleEnergyY = -1

	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y

		//TODO: This move logic may overwrite stuff for larger cells
		//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Checking if can move from %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

		if WS.CanMoveHere(cell, xTry, yTry, cell.Z) {
			//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Move from %d, %d, %d -> %d, %d, %d is possible\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

			var coveringCellInThisLocation, hasCoveringCell = WS.GetCoveringCellAt(xTry, yTry, cell.Z)
			if hasCoveringCell {
				//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: %d, %d, %d is covered by cell %d with %6.1f energy\n", cell.ID, xTry, yTry, cell.Z, coveringCellInThisLocation.ID, coveringCellInThisLocation.Energy)
				if coveringCellInThisLocation.Energy > highestEdibleEnergy {
					foundSomewhereWithEnergy = true
					//			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: %d, %d, %d has highest edible energy so far (%6.1f)\n", cell.ID, xTry, yTry, cell.Z, coveringCellInThisLocation.Energy)

					highestEdibleEnergy = coveringCellInThisLocation.Energy
					highestEdibleEnergyX = coveringCellInThisLocation.X
					highestEdibleEnergyY = coveringCellInThisLocation.Y
				}
			} else {
				//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: %d, %d, %d has NO covering cell\n", cell.ID, xTry, yTry, cell.Z)
			}

			//	Log(LOGTYPE_CELLEFFECT, "cell %d: Moving %d, %d, %d -> %d, %d, %d\n", cell.ID, cell.X, cell.Y, cell.Z, xTry, yTry, cell.Z)

		}
	}

	if foundSomewhereWithEnergy {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Location %d, %d, %d had highest energy (%6.1f). Moving there\n", cell.ID, highestEdibleEnergyX, highestEdibleEnergyY, cell.Z, highestEdibleEnergy)
		cell.moveHere(highestEdibleEnergyX, highestEdibleEnergyY)
		return true
	} else {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Unable to find cell with energy surrounding %d, %d, %d. Attempting to move randomly\n", cell.ID, cell.X, cell.Y, cell.Z)
		cell.MoveRandom()
		return false
	}
}

func (cell *Cell) Wait() {
	//if !cell.isDead() {
	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Waiting %d -> %d\n", cell.ID, cell.TimeLeftToWait, cell.ClockRate*ACTUAL_WAIT_MULTIPLIER)
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

func (cell *Cell) IsReadyToGrowDigestiveSystem() bool {
	return cell.DigestiveSystem == false && cell.Legs == true && cell.Height >= 1 && cell.Energy > cell.GrowDigestiveSystemAt
}

func (cell *Cell) IsReadyToGrowHeight() bool {
	//TODO: Update when you can grow height beyond 1
	return cell.Height == 0 && cell.Energy > cell.GrowHeightAt && !WS.IsSolidOrCovered(cell.X, cell.Y, cell.Z+cell.Height+1)
}

func (cell *Cell) WantsToAndCanEat() bool {
	if cell.DigestiveSystem == false || cell.Energy < EAT_COST {
		return false
	}

	//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MADE IT PAST FIRST EAT CHECK\n", cell.ID)
	var isCellEatableInLocation = WS.IsCovered(cell.X, cell.Y, cell.Z)
	if isCellEatableInLocation {
		//TODO: This should be more up to cell in future
		//TODO: This could be combined with check above
		//TODO: Will need to reflect 'eatable cell' nature here, vs. just covering, eventually
		var eatableCellInLocation, _ = WS.GetCoveringCellAt(cell.X, cell.Y, cell.Z)
		//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MADE IT PAST SECOND EAT CHECK\n", cell.ID)
		if eatableCellInLocation.Energy*FRACTION_EATEN_PER_EAT > EAT_COST+10 {
			//			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: WILL EAT (bite energy %6.1f is enough)\n", cell.ID, eatableCellInLocation.Energy*FRACTION_EATEN_PER_EAT)
			return true
		} else {
			//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: WILL NOT EAT (bite energy %6.1f is NOT enough)\n", cell.ID, eatableCellInLocation.Energy*FRACTION_EATEN_PER_EAT)
			return false
		}
	} else {
		return false
	}
}

func (cell *Cell) CanMove() bool {
	if cell.isDead() || cell.Legs == false {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE LOCATION STATUS %d, %d, %d failed: no legs or is dead\n", cell.ID, cell.X, cell.Y, cell.Z)
		return false
	}

	var isThereASpotToMove = false
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = cell.X + relativeX
			var yTry = cell.Y + relativeY
			if WS.CanMoveHere(cell, xTry, yTry, cell.Z) {
				isThereASpotToMove = true
				goto foundSpot
			}
		}
	}

foundSpot:
	if !isThereASpotToMove {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE LOCATION STATUS %d, %d, %d failed: no location to move to nearby\n", cell.ID, cell.X, cell.Y, cell.Z)

		return false
	}
	return cell.Energy > cell.GetMoveCost()
}

func (cell *Cell) GetMoveCost() float64 {
	const HEIGHT_MOVE_COST = .15
	const LEGS_MOVE_COST = .05
	const DIGESTIVESYSTEM_MOVE_COST = .0125
	const BASIC_BRAIN_MOVE_COST = .0125
	const CANOPY_MOVE_COST = .05

	var totalMoveCost = 0.0
	if cell.Height > 0 {
		totalMoveCost += HEIGHT_MOVE_COST * float64(cell.Height)
	}
	if cell.Legs {
		totalMoveCost += LEGS_MOVE_COST
	}
	if cell.DigestiveSystem {
		totalMoveCost += DIGESTIVESYSTEM_MOVE_COST
	}
	//TODO: Change when brain is changeable
	if true {
		totalMoveCost += BASIC_BRAIN_MOVE_COST
	}
	if cell.Canopy {
		totalMoveCost += CANOPY_MOVE_COST
	}

	if cell.Chloroplasts {
		totalMoveCost += CHLOROPLAST_MOVE_COST
	}

	return totalMoveCost
}

func (cell *Cell) WantsToAndCanMove() bool {
	//TODO: Do we need to be doing this on each pre-check?
	if cell.isDead() || cell.Legs == false {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE LOCATION STATUS %d, %d, %d failed: no legs or is dead\n", cell.ID, cell.X, cell.Y, cell.Z)
		return false
	}

	if float64(rand.Intn(100)) > cell.MoveChance {
		//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: doesn't want to move (failed movechance check)\n", cell.ID)
		return false
	}

	return cell.CanMove()
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
		//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Growing Canopy\n", cell.ID)
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
