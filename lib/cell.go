package lib

import "math/rand"

//---UKEEEP-------------
const CELL_LIFESPAN = 300

const BASE_CELL_UPKEEP = 1.0
const CANOPY_UPKEEP = 4.0 * BASE_CELL_UPKEEP
const LEGS_UPKEEP = .2

//---ACTION_COSTS-------
const MOVE_COST = 5
const THINKING_COST = 5.0
const REPRODUCE_COST = 30
const GROWCANOPY_COST = 4.0 * REPRODUCE_COST
const GROWLEGS_COST = 45

//TODO: Make it easy to add a field and have it appear in all the right places Re: copying and whatnot
type Cell struct {
	Energy                             float64
	X                                  int
	Y                                  int
	ID                                 int
	TimeLeftToWait                     int
	ClockRate                          int
	Canopy                             bool
	Legs                               bool
	GrowCanopyAt                       float64
	GrowLegsAt                         float64
	EnergySpentOnReproducing           float64
	NextMomentSelf                     *Cell
	Age                                int
	SpeciesID                          int
	PercentChanceWait                  int     //out of 100
	MoveChance                         float64 //out of 100
	_secretID                          int
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

func (cell *Cell) DecreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	if (!(cell.Energy <= 0.0)) || cell.NextMomentSelf != nil {
		//	if !cell.isDead() {
		//Log(LOGTYPE_CELLEFFECT, "the future cell %d currently has %f energy\n", cell.NextMomentSelf.ID, cell.NextMomentSelf.Energy)
		//Log(LOGTYPE_CELLEFFECT, "decreased cell %d by %f\n", cell.NextMomentSelf.ID, amt)
		cell.NextMomentSelf.Energy = cell.NextMomentSelf.Energy - amt
		//Log(LOGTYPE_CELLEFFECT, "cell %d will now have %f energy\n", cell.NextMomentSelf.ID, cell.NextMomentSelf.Energy)
	}
}

func (cell *Cell) IncreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	if (!(cell.Energy <= 0.0)) || cell.NextMomentSelf != nil {
		//	Log(LOGTYPE_CELLEFFECT, "the future cell %d currently has %f energy\n", cell.NextMomentSelf.ID, cell.NextMomentSelf.Energy)
		//	Log(LOGTYPE_CELLEFFECT, "increased cell %d by %f\n", cell.NextMomentSelf.ID, amt)
		cell.NextMomentSelf.Energy = cell.NextMomentSelf.Energy + amt

		//	Log(LOGTYPE_CELLEFFECT, "cell %d will now have %f energy\n", cell.NextMomentSelf.ID, cell.NextMomentSelf.Energy)
	}
}

func (cell *Cell) Maintain() {
	var totalUpkeep = BASE_CELL_UPKEEP
	if cell.Legs {
		totalUpkeep += LEGS_UPKEEP
	}
	if cell.Canopy {
		totalUpkeep += CANOPY_UPKEEP
	}
	//	Log(LOGTYPE_CELLEFFECT, "cell %d is about to be maintained\n", cell.ID)
	cell.DecreaseEnergy((totalUpkeep * float64(cell.Age)) / CELL_LIFESPAN)
	//cell.DecreaseEnergy(40)
	//	cell.Energy -= totalUpkeep * float64(cell.Age) / CELL_LIFESPAN
	cell.increaseAge(1)
}

func (cell *Cell) increaseAge(amt int) {
	if !cell.isDead() {
		cell.NextMomentSelf.Age = cell.NextMomentSelf.Age + amt
	}
}

func (cell *Cell) IncreaseWaitTime(amt int) {
	if !cell.isDead() {
		cell.NextMomentSelf.TimeLeftToWait += amt
	}
}

func (cell *Cell) GrowLegs() {

	if cell.isDead() {
		return
	} else if !cell.IsReadyToGrowLegs() {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.NextMomentSelf.DecreaseEnergy(GROWLEGS_COST)
		return
	} else {
		var NextMomentSelf = cell.NextMomentSelf
		//TODO: Disabled leg growing. Should re-enable when it's no longer under suspicion
		NextMomentSelf.Legs = false
		NextMomentSelf.DecreaseEnergy(GROWLEGS_COST)
	}
}

func moveRandom(cell *Cell) bool {
	if !cell.WantsToAndCanMove() {
		return false
	}

	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y

		//TODO: Really need to do some locking here to prevent corruption
		if !NextMoment.IsOccupied(xTry, yTry) {
			NextMoment.CellsSpatialIndex[xTry][yTry] = cell
			NextMoment.CellsSpatialIndex[cell.X][cell.Y] = nil
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
		cell.NextMomentSelf.TimeLeftToWait = ACTUAL_WAIT_MULTIPLIER * cell.ClockRate
	}
}

func (cell *Cell) ShouldWait() bool {
	return rand.Intn(100) < cell.PercentChanceWait
}

func (cell *Cell) isDead() bool {
	//TODO: Why do I have to check future self as well?

	return cell.Energy <= 0 || cell.NextMomentSelf == nil
}

func (cell *Cell) IsTimeToReproduce() bool {
	var isThereASpotToReproduce = false
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
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

func (cell *Cell) WantsToAndCanMove() bool {
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
	return cell.Legs == false && cell.Energy > cell.GrowLegsAt
}

func (cell *Cell) CountDown_TimeLeftToWait() {
	if !cell.isDead() {
		cell.NextMomentSelf.TimeLeftToWait--
	}
}

func (cell *Cell) GrowCanopy() {
	if cell.isDead() {
		return
	} else if cell.Energy < GROWCANOPY_COST {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.DecreaseEnergy(GROWCANOPY_COST)
		return
		//TODO: Not sure why I need to check this condition twice, but it seems to prevent a nil reference thing
	} else if !cell.isDead() {
		cell.NextMomentSelf.Canopy = true
		//TODO: Reinstate this log
		//Log(LOGTYPE_CELLEFFECT, "Cell %d grew canopy\n", cell.ID)
		cell.NextMomentSelf.DecreaseEnergy(GROWCANOPY_COST)
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

//-----CELL POOL--------
