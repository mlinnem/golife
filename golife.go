package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/davecheney/profile"
	. "github.com/mlinnem/golife/main/lib"
)

const RANDOM_SEED = true

const MAX_MOMENTS = 50000000

var HIGH_MUTATION_CHANCE = 10
var NO_MUTATION_CHANCE = 80

var CHANCE_OF_SANE_PLANT_VALUE = 0
var CHANCE_OF_RIGGED_ANIMAL = -1

//---PERFORMANCE_VARIABLES---
//--GENERAL---

//NOTE: If this is on, CellActionDecider can't be parallel
const SERIAL_ACTIONTOEXECUTION_BRIDGE = true
const DEFAULT_BRIDGE_ACTIONCAPACITY = MAX_CELL_COUNT

const PERSISTENT_CELLACTIONDECIDER = false
const PARALLELIZE_DECIDER = false
const CELLACTIONDECIDER_ROUTINECOUNT = 1
const CELLS_PER_CELLDECIDERBUNDLE = 1000

const PERSISTENT_CELLACTIONEXECUTER = false
const PARALLELIZE_CELLEXECUTER = false //Probably not good to say true for thread-safety reasons
const CELLACTIONEXECUTER_ROUTINECOUNT = 1

const PERSISTENT_NONCELLACTIONEXECUTER = false
const PARALLELIZE_NONCELLEXECUTER = false //Probably not good to say true for thread-safety reasons
const NONCELLACTIONEXECUTER_ROUTINECOUNT = 1

//--SPECIFIC--
const PARALLELIZE_SHINE_BY_ROW = true
const SHINE_FREQUENCY = 1

var cellActionExecuterWG sync.WaitGroup

var cellsReadyToDecide = make(chan []*Cell, MAX_CELL_COUNT)

var queuedNonCellActions []*NonCellAction
var nonCellActionsReadyToExecute = make(chan *NonCellAction, MAX_CELL_COUNT)

var serialQueuedCellActionsLength = 0
var serialQueuedCellActions []*CellAction
var queuedCellActions = make(chan *CellAction, MAX_CELL_COUNT)
var cellActionsReadyToExecute = make(chan *CellAction, MAX_CELL_COUNT)

var cellActionDeciderWG sync.WaitGroup
var nonCellActionExecuterWG sync.WaitGroup

var waitForCleaning sync.WaitGroup

var WSNum = 0

func startPersistentGoRoutines() {
	if PERSISTENT_CELLACTIONDECIDER {
		startCellActionDeciders()
	}
	if PERSISTENT_CELLACTIONEXECUTER {
		startCellActionExecuters()
	}
	if PERSISTENT_NONCELLACTIONEXECUTER {
		startNonCellActionExecuters()
	}
}

func startCellActionDeciders() {
	for i := 0; i < CELLACTIONDECIDER_ROUTINECOUNT; i++ {
		go cellActionDecider()
	}
}

func startCellActionExecuters() {
	for i := 0; i < CELLACTIONEXECUTER_ROUTINECOUNT; i++ {
		go cellActionExecuter()
	}
}

func startNonCellActionExecuters() {
	for i := 0; i < NONCELLACTIONEXECUTER_ROUTINECOUNT; i++ {
		go nonCellActionExecuter(&nonCellActionExecuterWG)
	}
}

func removeDeadCells() {
	Log(LOGTYPE_MAINLOOPSINGLE, "started turn with %d cells\n", len(WS.Cells))
	for i := len(WS.Cells) - 1; i >= 0; i-- {
		cell := WS.Cells[i]
		Log(LOGTYPE_HIGHFREQUENCY, "a cell %d, has %6.2f energy\n", cell.ID, cell.Energy)
		if cell.Energy <= 0 {
			WS.Cells = append(WS.Cells[:i],
				WS.Cells[i+1:]...)
			WS.RemoveCellFromSpatialIndex(cell)
		}
	}
	Log(LOGTYPE_MAINLOOPSINGLE, " %d cells after culling \n", len(WS.Cells))
	updateTracedCell()
}

func sendAllCellsToDecider() {
	if SERIAL_ACTIONTOEXECUTION_BRIDGE {
		//	serialQueuedCellActions = make([]*CellAction, 0, DEFAULT_BRIDGE_ACTIONCAPACITY)
		serialQueuedCellActionsLength = 0
	}

	var startSlice = 0
	var endSlice = CELLS_PER_CELLDECIDERBUNDLE
	var cellsToSend []*Cell
	for {
		if endSlice < len(WS.Cells) {
			cellsToSend = WS.Cells[startSlice:endSlice]
			sendCellSliceToDecider(cellsToSend)
			startSlice += CELLS_PER_CELLDECIDERBUNDLE
			endSlice += CELLS_PER_CELLDECIDERBUNDLE
		} else {
			cellsToSend = WS.Cells[startSlice:]
			sendCellSliceToDecider(cellsToSend)
			break
		}
	}
}

func sendCellSliceToDecider(cellsToSend []*Cell) {
	if PERSISTENT_CELLACTIONDECIDER {
		cellActionDeciderWG.Add(1)
		cellsReadyToDecide <- cellsToSend
	} else if PARALLELIZE_DECIDER {
		cellActionDeciderWG.Add(1)
		go decideForTheseCells(cellsToSend, true)
	} else {
		decideForTheseCells(cellsToSend, false)
	}
}

func sendAllCellActionsToExecuter() {
	if SERIAL_ACTIONTOEXECUTION_BRIDGE {
		for i := 0; i < serialQueuedCellActionsLength; i++ {
			sendSingleCellActionToExecuter(serialQueuedCellActions[i])
		}
	} else {
		queuedCellActions <- &CellAction{actionType: specialcellaction_done}
		for {
			var action = <-queuedCellActions
			//TODO: This could be more Golang idiomatic. Not sure if it affects performance though
			if action.actionType == specialcellaction_done {
				break
			} else {
				sendSingleCellActionToExecuter(action)
			}
		}
	}
}

func sendSingleCellActionToExecuter(action *CellAction) {
	if PERSISTENT_CELLACTIONEXECUTER {
		cellActionExecuterWG.Add(1)
		cellActionsReadyToExecute <- action
	} else if PARALLELIZE_CELLEXECUTER {
		cellActionExecuterWG.Add(1)
		go executeSingleCellAction(action, true)
	} else {
		executeSingleCellAction(action, false)
	}
}

func sendAllNonCellActionsToExecuter() {
	for ai := 0; ai < len(queuedNonCellActions); ai++ {
		var nonCellAction = queuedNonCellActions[ai]
		if PERSISTENT_NONCELLACTIONEXECUTER {
			nonCellActionExecuterWG.Add(1)
			nonCellActionsReadyToExecute <- nonCellAction
		} else if PARALLELIZE_NONCELLEXECUTER {
			nonCellActionExecuterWG.Add(1)
			go executeSingleNonCellAction(nonCellAction, true)
		} else {
			executeSingleNonCellAction(nonCellAction, false)
		}
	}
	//empty queue now that they've been sent. Probably better way to do this
	queuedNonCellActions = []*NonCellAction{}
}

func updateTracedCell() {
	if TracedCell != nil && TracedCell.Energy <= 0 {
		TracedCell = nil
	}
	if TracedCell == nil && len(WS.Cells) > 0 {
		for multiplier := 1; multiplier < TRACEDCELL_AGERANGE_EXPANSIONS; multiplier++ {
			for _, cell := range WS.Cells {
				if cell.Age < TRACEDCELL_AGECAP*multiplier {
					TracedCell = cell
					return
				}
			}
		}
		Log(LOGTYPE_CELLEFFECT, "XXXXXXXX - Unable to find cell to trace with age < %d. Picking first cell from list\n", TRACEDCELL_AGECAP*TRACEDCELL_AGERANGE_EXPANSIONS)
		TracedCell = WS.Cells[0]
	}
}

func generateSystemNonCellActions() {
	//generate any nonCellActions that need to be generated
	//TODO: Rigging to try to introduce some animals later on
	if WSNum == 0 {
		Log(LOGTYPE_MAINLOOPSINGLE, "Populating initial cells\n")
		generateInitialNonCellActions(&nonCellActionExecuterWG)
		//	close(queuedNonCellActions)
	} else {
		Log(LOGTYPE_MAINLOOPSINGLE, "Generating cell maintenance and other stuff\n")
		generateSunshineAction(&nonCellActionExecuterWG)
		generateCellMaintenanceAction(&nonCellActionExecuterWG)
		//TODO: Testing reap removal
		//generateGrimReaperAction(&nonCellActionExecuterWG)
	}
}

func initializeState() {
	WS = &WorldState{}

	startPersistentGoRoutines()

	if SERIAL_ACTIONTOEXECUTION_BRIDGE {
		serialQueuedCellActionsLength = 0
		serialQueuedCellActions = make([]*CellAction, DEFAULT_BRIDGE_ACTIONCAPACITY, DEFAULT_BRIDGE_ACTIONCAPACITY)
	}
}

func main() {
	if RANDOM_SEED {
		rand.Seed(int64(time.Now().Second()))
	}
	defer profile.Start(profile.CPUProfile).Stop()

	var startTime = time.Now()

	initializeState()

	for WSNum = 0; WSNum < MAX_MOMENTS; WSNum++ {
		WS.WSNum = WSNum

		if WSNum%PRINTGRID_EVERY_N_TURNS == 0 {
			PrintGrid(WS, DEFAULT_PRINTGRID_DEPTH)
			PrintSpeciesReport(WS, NUM_TOP_SPECIES_TO_PRINT)
		}
		Log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "WS %d...\n", WSNum)

		Log(LOGTYPE_MAINLOOPSINGLE, "Start of turn upkeep ...\n", WSNum)

		removeDeadCells()

		Log(LOGTYPE_MAINLOOPSINGLE, "Start deciding...\n")

		sendAllCellsToDecider()

		Log(LOGTYPE_MAINLOOPSINGLE, "Waiting for decisions to finish ...\n")
		cellActionDeciderWG.Wait()
		Log(LOGTYPE_MAINLOOPSINGLE, "Decisions finished.\n")

		sendAllCellActionsToExecuter()

		Log(LOGTYPE_MAINLOOPSINGLE, "Waiting for execution to finish...\n")
		cellActionExecuterWG.Wait()
		Log(LOGTYPE_MAINLOOPSINGLE, "Execution finished.\n")
		//}

		Log(LOGTYPE_MAINLOOPSINGLE, "Generating System NonCellActions...\n")
		generateSystemNonCellActions()
		Log(LOGTYPE_MAINLOOPSINGLE, "Generating NonCellActions Finished.\n")

		Log(LOGTYPE_MAINLOOPSINGLE, "Feeding NonCellActions to nonCellActionExecuter(if any)\n")
		sendAllNonCellActionsToExecuter()

		Log(LOGTYPE_MAINLOOPSINGLE, "Waiting for NonCellActionExecuter to finish...\n")
		nonCellActionExecuterWG.Wait()
		Log(LOGTYPE_MAINLOOPSINGLE, "Non-cell Executers did their thing\n")

		Log(LOGTYPE_MAINLOOPSINGLE, "Moment finished\n")

		if len(WS.Cells) == 0 {
			Log(LOGTYPE_FAILURES, "Early termination due to all cells dying\n")
			break
		}
	}

	Log(LOGTYPE_FINALSTATS, "%d cells in final WS\n", len(WS.Cells))

	var endTime = time.Now()
	var fullDuration = endTime.Sub(startTime).Seconds()
	Log(LOGTYPE_FINALSTATS, "Time of entire run took %f\n", fullDuration)
}

//CELL-ACTION DECIDER

type CellAction struct {
	cell       *Cell
	actionType int
}

const (
	cellaction_growcanopy                = iota
	cellaction_growheight                = iota
	cellaction_growlegs                  = iota
	cellaction_growdigestivesystem       = iota
	cellaction_growchloroplasts          = iota
	cellaction_wait                      = iota
	cellaction_reproduce                 = iota
	cellaction_moverandom                = iota
	cellaction_movetohighestedibleenergy = iota
	cellaction_eat                       = iota
	specialcellaction_done               = iota
)

func getCellActionName(cellActionType int) string {
	switch cellActionType {
	case cellaction_reproduce:
		return "Reproduce"
	case cellaction_growcanopy:
		return "Grow Canopy"
	case cellaction_growheight:
		return "Grow Height"
	case cellaction_wait:
		return "Wait"
	case cellaction_eat:
		return "Eat"
	case cellaction_growlegs:
		return "Grow Legs"
	case cellaction_movetohighestedibleenergy:
		return "Move to Highest Edible Energy"
	case cellaction_moverandom:
		return "Move Random"
	case cellaction_growchloroplasts:
		return "Grow Chloroplasts"
	case cellaction_growdigestivesystem:
		return "Grow Digestive System"
	default:
		return "NO_KNOWN_ACTION_NAME"
	}

}

//A bundle of length 0 is the signal to stop
func cellActionDecider() {
	for {
		var cellDeciderBundle = <-cellsReadyToDecide
		decideForTheseCells(cellDeciderBundle, true)
	}
}

func decideForTheseCells(cellDeciderBundle []*Cell, asynchronous bool) {
	for _, cell := range cellDeciderBundle {
		if cell.TimeLeftToWait > 0 {
			cell.CountDown_TimeLeftToWait()
		} else {
			var cellAction = decideSingleCell(cell)
			if cellAction != nil {
				//			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: sending action '%s'\n", cell.ID, getCellActionName(cellAction.actionType))
				//TODO: Is this safe when executer is persistent?
				if SERIAL_ACTIONTOEXECUTION_BRIDGE {
					serialQueuedCellActions[serialQueuedCellActionsLength] = cellAction
					serialQueuedCellActionsLength++
				} else {
					queuedCellActions <- cellAction
				}
			} else {
				//		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: NO ACTION\n", cell.ID)
			}
		}
	}
	if asynchronous {
		cellActionDeciderWG.Done()
	}
}

func decideSingleCell(cell *Cell) *CellAction {
	var cellAction *CellAction
	if cell.IsReadyToGrowHeight() {
		cellAction = &CellAction{cell, cellaction_growheight}
	} else if cell.IsReadyToGrowDigestiveSystem() {
		cellAction = &CellAction{cell, cellaction_growdigestivesystem}
	} else if cell.IsReadyToGrowCanopy() {
		cellAction = &CellAction{cell, cellaction_growcanopy}
	} else if cell.IsReadyToGrowLegs() {
		cellAction = &CellAction{cell, cellaction_growlegs}
	} else if cell.IsReadyToGrowChloroplasts() {
		cellAction = &CellAction{cell, cellaction_growchloroplasts}
	} else if cell.IsTimeToReproduce() {
		cellAction = &CellAction{cell, cellaction_reproduce}
	} else if cell.WantsToAndCanEat() {
		cellAction = &CellAction{cell, cellaction_eat}
	} else if cell.WantsToAndCanMove() {
		//TODO: Rigged to move to highest energy, despite move random logic in "Wants To And Can Move"
		cellAction = &CellAction{cell, cellaction_movetohighestedibleenergy}
	} else if cell.ShouldWait() {
		cellAction = &CellAction{cell, cellaction_wait}
	} else {
		//no action at all. Hopefully don't need to submit a null action
	}

	//LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: PAY THINKING\n", cell.ID)
	cell.DecreaseEnergy(THINKING_COST)
	cell.IncreaseWaitTime(cell.ClockRate - 1)
	return cellAction
}

func NilChecker(slice []*CellAction, stamp string) {
	for i, element := range slice {
		if element == nil {
			log.Printf("Element %d is nil in detectotron: %s\n", i, stamp)
			QueuedCellAction_Printer(slice, "EMERGENCY CHECK")
		}
	}
}

func QueuedCellAction_Printer(slice []*CellAction, stamp string) {
	log.Printf("%s\n", stamp)
	for i, element := range slice {
		if element == nil {
			log.Printf("\tElement %d is nil in detectotron: %s\n", i, stamp)
		}
		if element != nil {
			log.Printf("\tCellAction %d is for Cell %d, which as %f energy: Action is %s...(%s)\n", i, element.cell.ID, element.cell.Energy, getCellActionName(element.actionType), stamp)
		}
	}
}

//CELL-ACTION EXECUTER

func cellActionExecuter() {
	for {
		var cellAction = <-cellActionsReadyToExecute
		executeSingleCellAction(cellAction, true)
	}
}

func executeSingleCellAction(cellAction *CellAction, asynchronous bool) {
	//TODO: Shouldn't be getting nil cell actions. Figure out what's up
	//if cellAction == nil {
	//	return
	//}

	var cell = cellAction.cell

	switch cellAction.actionType {
	case cellaction_reproduce:
		reproduce(cellAction.cell)
	case cellaction_growdigestivesystem:
		cell.GrowDigestiveSystem()
	case cellaction_growcanopy:
		cell.GrowCanopy()
	case cellaction_growheight:
		cell.GrowHeight()
	case cellaction_growchloroplasts:
		cell.GrowChloroplasts()
	case cellaction_wait:
		cell.Wait()
	case cellaction_eat:
		cell.Eat()
	case cellaction_growlegs:
		cell.GrowLegs()
	case cellaction_moverandom:
		cell.MoveRandom()
	case cellaction_movetohighestedibleenergy:
		cell.MoveToHighestEdibleEnergy()
	}

	if asynchronous {
		cellActionExecuterWG.Done()
	}
}

//NON-CELL-ACTIONS

type NonCellAction struct {
	actionType int
}

const (
	noncellaction_spontaneouslyPlaceCell = iota
	noncellaction_shineOnAllCells        = iota
	noncellaction_cellMaintenance        = iota
)

func generateInitialNonCellActions(wg *sync.WaitGroup) {
	for i := 0; i < INITIAL_CELL_COUNT; i++ {
		//TODO: is this efficient? Maybe get rid of struct and use raw consts
		queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_spontaneouslyPlaceCell})
	}
}

func generateSunshineAction(wg *sync.WaitGroup) {
	//TODO for now this action is atom. Could break it up later.
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_shineOnAllCells})
}

func generateCellMaintenanceAction(wg *sync.WaitGroup) {
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_cellMaintenance})
}

func nonCellActionExecuter(wg *sync.WaitGroup) {
	for {
		var nonCellAction = <-nonCellActionsReadyToExecute
		executeSingleNonCellAction(nonCellAction, true)
	}
}

func executeSingleNonCellAction(nonCellAction *NonCellAction, asynchronous bool) {
	//route it to function depending on its type
	switch nonCellAction.actionType {
	case noncellaction_spontaneouslyPlaceCell:
		spontaneouslyGenerateCell()
	case noncellaction_shineOnAllCells:
		shineOnAllCells()
	case noncellaction_cellMaintenance:
		//TODO
		maintainAllCells()
	}

	if asynchronous {
		nonCellActionExecuterWG.Done()
	}
}

func mutateNormMax(magnitudeNorm float64, magnitudeMax float64) float64 {
	var rand0To99 = rand.Intn(100)

	if rand0To99 < HIGH_MUTATION_CHANCE {
		return float64(rand.Intn(int(magnitudeMax*2.0))) - magnitudeMax
	} else if rand0To99 < 100-NO_MUTATION_CHANCE {
		return float64(rand.Intn(int(magnitudeNorm*2.0))) - magnitudeNorm
	} else {
		return 0.0
	}
}

func boundMinMax(current float64, min float64, max float64) float64 {
	return math.Min(math.Max(current, min), max)
}

func mutateMinMaxInThisRange(current float64, magnitudeNorm float64, magnitudeMax float64, min float64, max float64) float64 {
	return boundMinMax(current+mutateNormMax(magnitudeNorm, magnitudeMax), min, max)
}

//TODO: Should be in cell.go?
func reproduce(cell *Cell) {
	if cell.Energy < REPRODUCE_COST || cell.Energy < cell.EnergySpentOnReproducing {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.DecreaseEnergy(cell.EnergySpentOnReproducing)
		return
	}
	//try all spots surrounding the cell
	var z = 0
	//TODO: Need to figure out how to handle bunnies reproducing on top of plants. Should be possible.
	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y
		if !WS.IsSolidOrCovered(xTry, yTry, z) {
			//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Making baby from %d, %d -> %d, %d\n", cell.ID, cell.X, cell.Y, xTry, yTry)
			var babyCell = CellPool.Borrow()
			babyCell.Energy = cell.EnergySpentOnReproducing - REPRODUCE_COST
			//TODO: This should be a function, probably needs locking if parallelized
			babyCell.ID = IDCounter
			IDCounter++
			//	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d born with %f energy\n", babyCell.ID, babyCell.Energy)
			babyCell.X = xTry
			babyCell.Y = yTry
			babyCell.Z = 0
			babyCell.SpeciesID = cell.SpeciesID
			babyCell.SpeciesColor = cell.SpeciesColor
			babyCell.TimeLeftToWait = 0
			babyCell.Age = 0
			babyCell.Canopy = false
			babyCell.Height = 0
			babyCell.Legs = false
			babyCell.Chloroplasts = false
			babyCell.DigestiveSystem = false

			babyCell.ClockRate = int(mutateMinMaxInThisRange(float64(cell.ClockRate), 5.0, 50.0, 1.0, 1000.0))
			babyCell.EnergySpentOnReproducing = mutateMinMaxInThisRange(cell.EnergySpentOnReproducing, 8.0, 1000.0, REPRODUCE_COST, 9000.0)
			babyCell.EnergyReproduceThreshold = mutateMinMaxInThisRange(cell.EnergyReproduceThreshold, 8.0, 1000.0, babyCell.EnergySpentOnReproducing, 9000.0)
			babyCell.GrowChloroplastsAt = mutateMinMaxInThisRange(cell.GrowChloroplastsAt, 8.0, 1000.0, GROWCHLOROPLASTS_COST, 9000.0)
			babyCell.GrowCanopyAt = mutateMinMaxInThisRange(cell.GrowCanopyAt, 8.0, 1000.0, GROWCANOPY_COST, 9000.0)
			babyCell.GrowDigestiveSystemAt = mutateMinMaxInThisRange(cell.GrowDigestiveSystemAt, 8.0, 1000.0, GROWDIGESTIVESYSTEM_COST, 9000.0)
			babyCell.GrowHeightAt = mutateMinMaxInThisRange(cell.GrowHeightAt, 8.0, 1000.0, GROWHEIGHT_COST, 9000.0) //math.Max(GROWHEIGHT_COST, cell.GrowHeightAt+float64(rand.Intn(9)-5))
			babyCell.GrowLegsAt = mutateMinMaxInThisRange(cell.GrowLegsAt, 8.0, 1000.0, GROWLEGS_COST, 9000.0)
			babyCell.MoveChance = mutateMinMaxInThisRange(cell.MoveChance, 6.0, 100.0, 0, 100.0)
			babyCell.PercentChanceWait = int(mutateMinMaxInThisRange(float64(cell.PercentChanceWait), 6.0, 100.0, 0, 100.0))

			babyCell.X_originalClockRate = cell.X_originalClockRate
			babyCell.X_originalEnergyReproduceThreshold = cell.X_originalEnergyReproduceThreshold
			babyCell.X_originalEnergySpentOnReproducing = cell.X_originalEnergySpentOnReproducing
			babyCell.X_originalGrowCanopyAt = cell.X_originalGrowCanopyAt
			babyCell.X_originalGrowChloroplastsAt = cell.X_originalGrowChloroplastsAt
			babyCell.X_originalGrowDigestiveSystemAt = cell.X_originalGrowDigestiveSystemAt
			babyCell.X_originalGrowHeightAt = cell.X_originalGrowHeightAt
			babyCell.X_originalGrowLegsAt = cell.X_originalGrowLegsAt
			babyCell.X_originalMoveChance = cell.X_originalMoveChance
			babyCell.X_originalPercentChanceWait = cell.X_originalPercentChanceWait

			//TODO: WTF this shit is hella bugged in some mysterious way
			if HasSignificantGeneticDivergence(babyCell) {
				//TODO: This should probably be a function
				babyCell.SpeciesID = SpeciesIDCounter
				SpeciesIDCounter++
				babyCell.SpeciesColor = getNextColor()

				babyCell.X_originalClockRate = babyCell.ClockRate
				babyCell.X_originalEnergySpentOnReproducing = babyCell.EnergySpentOnReproducing
				babyCell.X_originalEnergyReproduceThreshold = babyCell.EnergyReproduceThreshold
				babyCell.X_originalGrowChloroplastsAt = babyCell.GrowChloroplastsAt
				babyCell.X_originalGrowCanopyAt = babyCell.GrowCanopyAt
				babyCell.X_originalGrowDigestiveSystemAt = babyCell.GrowDigestiveSystemAt
				babyCell.X_originalGrowHeightAt = babyCell.GrowHeightAt
				babyCell.X_originalGrowLegsAt = babyCell.GrowLegsAt
				babyCell.X_originalMoveChance = babyCell.MoveChance
				babyCell.X_originalPercentChanceWait = babyCell.PercentChanceWait

				Log(LOGTYPE_SPECIALEVENT, "Cell at %d, %d "+babyCell.SpeciesColor.StartSequence+"x"+babyCell.SpeciesColor.EndSequence+" is the first of a new species!\n", xTry, yTry)
				Log(LOGTYPE_SPECIALEVENT, "orig reprod threshold: %f, new reprod threshold: %f\n", cell.X_originalEnergyReproduceThreshold, babyCell.X_originalEnergyReproduceThreshold)
				Log(LOGTYPE_SPECIALEVENT, "orig reprod energy spend: %f, new reprod energy spend: %f\n", cell.X_originalEnergySpentOnReproducing, babyCell.X_originalEnergySpentOnReproducing)
				Log(LOGTYPE_SPECIALEVENT, "orig grow canopy threshold: %f, new grow canopy threshold: %f\n", cell.X_originalGrowCanopyAt, babyCell.X_originalGrowCanopyAt)
			}
			WS.Cells = append(WS.Cells, babyCell)
			WS.AddCellToSpatialIndex(babyCell)
			if TracedCell == cell {
				TracedCell = babyCell
			}
			cell.DecreaseEnergy(cell.EnergySpentOnReproducing)
			return
		}
	}
	//	unlockYRangeInclusive(cell.Y-1, cell.Y+1, "reproduce")
}

func maintainAllCells() {
	Log(LOGTYPE_MAINLOOPSINGLE, "Starting maintain\n")
	//TODO: Could be parallelized in future
	for _, cell := range WS.Cells {
		cell.Maintain()
	}
	Log(LOGTYPE_MAINLOOPSINGLE, "Ending maintain\n")
}

var SpeciesIDCounter = 0
var IDCounter = 0

func spontaneouslyGenerateCell() {
	//TODO: Probably need to lock some stuff here
	//TODO: Cell pool should probaby zero out cell values before handing it off
	var newCell = CellPool.Borrow()
	var foundSpotYet = false
	var tries = 0
	var giveUp = false

	var z = 0

	for !foundSpotYet && !giveUp {

		var xTry = rand.Intn(GRID_WIDTH)
		var yTry = rand.Intn(GRID_HEIGHT)
		if !WS.IsSolidOrCovered(xTry, yTry, z) {
			foundSpotYet = true
			newCell.X = xTry
			newCell.Y = yTry
			newCell.SpeciesID = SpeciesIDCounter
			newCell.ID = IDCounter
			IDCounter++
			SpeciesIDCounter++
			newCell.SpeciesColor = getNextColor()
			newCell.Energy = float64(rand.Intn(4000))
			newCell.TimeLeftToWait = 0
			newCell.Chloroplasts = false

			if rand.Intn(100) < CHANCE_OF_SANE_PLANT_VALUE {
				newCell.ClockRate = rand.Intn(400) + 1
			} else {
				newCell.ClockRate = rand.Intn(45) + 5
			}

			newCell.PercentChanceWait = rand.Intn(90)

			if rand.Intn(100) < CHANCE_OF_SANE_PLANT_VALUE {
				newCell.EnergySpentOnReproducing = REPRODUCE_COST + float64(rand.Intn(4000))
			} else {
				newCell.EnergySpentOnReproducing = REPRODUCE_COST + float64(rand.Intn(100))
			}

			if rand.Intn(100) < CHANCE_OF_SANE_PLANT_VALUE {
				newCell.EnergyReproduceThreshold = newCell.EnergySpentOnReproducing + float64(rand.Intn(3000))
			} else {
				newCell.EnergyReproduceThreshold = newCell.EnergySpentOnReproducing + float64(rand.Intn(100))
			}

			newCell.Canopy = false
			newCell.GrowDigestiveSystemAt = float64(rand.Intn(4000)) + GROWDIGESTIVESYSTEM_COST

			if rand.Intn(100) < CHANCE_OF_SANE_PLANT_VALUE {
				newCell.GrowChloroplastsAt = float64(rand.Intn(4000)) + GROWCHLOROPLASTS_COST
			} else {
				newCell.GrowChloroplastsAt = float64(rand.Intn(30)) + GROWCHLOROPLASTS_COST
			}

			newCell.GrowCanopyAt = float64(rand.Intn(4000)) + GROWCANOPY_COST
			newCell.GrowHeightAt = float64(rand.Intn(4000)) + GROWHEIGHT_COST
			if rand.Intn(100) < CHANCE_OF_SANE_PLANT_VALUE {
				newCell.GrowLegsAt = float64(rand.Intn(4000)) + GROWLEGS_COST
				newCell.MoveChance = float64(rand.Intn(95))
			} else {
				newCell.GrowLegsAt = float64(rand.Intn(4000)) + 4000 + GROWLEGS_COST
				newCell.MoveChance = float64(rand.Intn(95))
			}

			//TODO: Straight up making an animal here, designer animal, only for testing purposes

			if rand.Intn(100) < CHANCE_OF_RIGGED_ANIMAL {
				newCell.Energy = 1000.0
				newCell.GrowDigestiveSystemAt = GROWDIGESTIVESYSTEM_COST + 50
				newCell.GrowChloroplastsAt = 5000.0
				newCell.GrowHeightAt = GROWHEIGHT_COST + 50
				newCell.MoveChance = 75
				newCell.GrowLegsAt = GROWLEGS_COST + 50
				newCell.PercentChanceWait = 90
				newCell.EnergyReproduceThreshold = 900.0
				newCell.EnergySpentOnReproducing = 800.0
				newCell.ClockRate = 1
				newCell.GrowCanopyAt = 5000.0
				TracedCell = newCell
			}

			newCell.X_originalGrowDigestiveSystemAt = newCell.GrowDigestiveSystemAt
			newCell.X_originalGrowChloroplastsAt = newCell.GrowChloroplastsAt
			newCell.X_originalGrowHeightAt = newCell.GrowHeightAt
			newCell.X_originalMoveChance = newCell.MoveChance
			newCell.X_originalGrowLegsAt = newCell.GrowLegsAt
			newCell.X_originalPercentChanceWait = newCell.PercentChanceWait
			newCell.X_originalEnergyReproduceThreshold = newCell.EnergyReproduceThreshold
			newCell.X_originalEnergySpentOnReproducing = newCell.EnergySpentOnReproducing
			newCell.X_originalClockRate = newCell.ClockRate
			newCell.X_originalGrowCanopyAt = newCell.GrowCanopyAt

			WS.Cells = append(WS.Cells, newCell)
			Log(LOGTYPE_HIGHFREQUENCY, "Added cell %d to next WS\n", newCell.ID)
			WS.AddCellToSpatialIndex(newCell)
		}
		tries++
		if tries > MAX_TRIES_TO_FIND_EMPTY_GRID_COORD {
			Log(LOGTYPE_FAILURES, "Gave up on placing tell. Too many cells occupied.")
			break
		}
	}
}

var colorCounter = 100
var textColorBookends = getAllTextColorBookends()

func getAllTextColorBookends() []*TextColorBookend {
	var textColorBookendsTemp []*TextColorBookend
	for i := 0; i <= 2; i++ {
		for k := 40; k <= 48; k++ {
			for j := 30; j <= 38; j++ {
				if (j + 10) == i {
					//fg on same bg, skip
					continue
				}
				var textColorStart = fmt.Sprintf("\033[%d;%d;%dm", i, j, k)
				var textColorEnd = "\033[m"
				//Log(LOGTYPE_FINALSTATS, textColorStart+" BLOOP "+textColorEnd)
				var textColorBookend = &TextColorBookend{textColorStart, textColorEnd}
				textColorBookendsTemp = append(textColorBookendsTemp, textColorBookend)
				//Log(LOGTYPE_FINALSTATS, "%d;%d;%d: \033[%d;%d;%dm Hello, World! \033[m \n", i, j, k, i, j, k)
			}
		}
	}
	return textColorBookendsTemp
}

func getNextColor() *TextColorBookend {
	//TODO this is a hack
	SpeciesCounter++
	var nextColor = textColorBookends[colorCounter]
	colorCounter++
	if colorCounter >= len(textColorBookends) {
		colorCounter = 0
	}
	return nextColor
}

//from https://gist.github.com/DavidVaini/10308388
func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func shineOnAllCells() {
	if WSNum%SHINE_FREQUENCY == 0 {
		Log(LOGTYPE_MAINLOOPSINGLE, "Starting shine\n")
		//TODO: This could stand to be refactored a bit
		//TODO: Changed for WS to current WS, but might be wrong...

		var isDayTime = WSNum%100 <= PERCENT_DAYLIGHT
		var wg = &sync.WaitGroup{}
		for yi := 0; yi < GRID_HEIGHT; yi++ {

			if PARALLELIZE_SHINE_BY_ROW {
				wg.Add(1)
				go shineThisRow(yi, isDayTime, wg)
			} else {
				shineThisRow(yi, isDayTime, wg)
			}

			//Log(LOGTYPE_HIGHFREQUENCY, "Shiner touching on %d, %d \n", xi, yi)
			//unlockYXRangeInclusive(yi-1, yi+1, xi-1, xi+1, "shiner")
			//TODO: May have placed lock/unlocks here incorrectly
		}
		Log(LOGTYPE_MAINLOOPSINGLE, "Ending shine\n")
		if PARALLELIZE_SHINE_BY_ROW {
			wg.Wait()
		}
	}

}

const SHADE_GRADIENT_OFFSET = 2

func shineThisRow(yi int, isDayTime bool, wg *sync.WaitGroup) {
	for xi := 0; xi < GRID_WIDTH; xi++ {
		//TODO: Disabled day night cycle. Turn it back on eventually
		if isDayTime { //int(YProximityToMiddleAsPercent(yi)*100)

			var shineAmountForThisSquare float64
			if xi%2 == 0 && yi%2 == 0 {
				shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * SHINE_FREQUENCY / (float64(PERCENT_DAYLIGHT) / 100.0) * SHADE_GRADIENT_OFFSET //* float64(float64(yi)/float64(GRID_HEIGHT))
			} else {
				shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * SHINE_FREQUENCY * (float64(xi) / GRID_WIDTH) / (float64(PERCENT_DAYLIGHT) / 100.0) * SHADE_GRADIENT_OFFSET //* float64(float64(yi)/float64(GRID_HEIGHT))
			}
			//fmt.Printf("Shining this much at %d, %d: %6.1f\n", xi, yi, shineAmountForThisSquare)
			newerShineMethod(xi, yi, shineAmountForThisSquare)
		} else {
			//No sun at night
		}
	}
	if PARALLELIZE_SHINE_BY_ROW {
		wg.Done()
	}
}

const SURROUNDINGS_SIZE = 9

func newerShineMethod(x int, y int, shineAmountForThisSquare float64) {
	//	fmt.Printf("Shinin \n")

	var shineAmountLeft = shineAmountForThisSquare
	var hitSolid = false
	for z := GRID_DEPTH - 1; z >= 0; z-- {
		//	fmt.Printf("Blam")
		if WS.SpatialIndexSolid[z][y][x] != nil {
			hitSolid = false
			//TODO: Rigged
		}

		var remaining = giveEnergyToNonSolidCellsAtThisLevel(x, y, z, shineAmountLeft*CANOPY_COVER_FRACTION)
		//TODO: This needs to handle roofs
		shineAmountLeft = shineAmountLeft/2 + remaining
		var cellAtSurface = WS.SpatialIndexSurfaceCover[z][y][x]

		if cellAtSurface != nil && !hitSolid && cellAtSurface.Chloroplasts == true {
			//fmt.Println("Giving juice to cell")
			//	LogIfTraced(cellAtSurface, LOGTYPE_CELLEFFECT, "cell %d: Surface shine @ height %d \n", cellAtSurface.ID, z)
			//fmt.Printf("cell %d: Surface shine of %6.1f @ height %d \n", cellAtSurface.ID, shineAmountLeft, z)
			//PrintCurrentGenesOfCell(cellAtSurface)

			//var cell = cellAtSurface
			//	Log(LOGTYPE_CELLEFFECT, "cell %d GROW STATUS: energy %6.1f, canopy %t, height %d, legs %t, chloroplasts %t, DigestiveSystem %t\n\n\n", cell.ID, cell.Energy, cell.Canopy, cell.Height, cell.Legs, cell.Chloroplasts, cell.DigestiveSystem)

			cellAtSurface.IncreaseEnergy(shineAmountLeft)
		}
	}
}

func giveEnergyToNonSolidCellsAtThisLevel(x int, y int, z int, shineAmountForThisSquare float64) float64 {
	//fmt.Printf("Shine on canopy folks at %d,%d,%d with %6.1f\n", x, y, z, shineAmountForThisSquare)
	var surroundingCellsWithCanopiesAndMe = &[SURROUNDINGS_SIZE]*Cell{} //surroundingsPool.Borrow()
	var numSurrounders = 0

	//var cell = WS.SpatialIndexSurfaceCover[z][y][x]

	for relativeY := -1; relativeY < 2; relativeY++ {
		for relativeX := -1; relativeX < 2; relativeX++ {
			var xTry = x + relativeX
			var yTry = y + relativeY

			if !WS.IsOutOfBounds(xTry, yTry, z) && WS.IsCovered(xTry, yTry, z) && WS.SpatialIndexSurfaceCover[z][yTry][xTry].Canopy == true {
				var surroundingCell = WS.SpatialIndexSurfaceCover[z][yTry][xTry]
				surroundingCellsWithCanopiesAndMe[numSurrounders] = surroundingCell
				numSurrounders++
			}
		}
	}
	var energyToEachCell = shineAmountForThisSquare / float64(numSurrounders)
	for i := 0; i < numSurrounders; i++ {
		//	LogIfTraced(surroundingCellsWithCanopiesAndMe[i], LOGTYPE_CELLEFFECT, "cell %d: Canopy shine @ height %d, 1/%d\n", surroundingCellsWithCanopiesAndMe[i].ID, z, numSurrounders)
		surroundingCellsWithCanopiesAndMe[i].IncreaseEnergy(energyToEachCell)
	}

	if numSurrounders == 0 {
		return shineAmountForThisSquare
	} else {
		return 0
	}
}

//func unlockYXRangeInclusive(startY int, endY int, startX int, endX int, who string) {
//	for y := startY; y < endY+1; y++ {
//		for x := startX; x < endX+1; x++ {
//			if !WS.IsOutOfBounds(x, y) {
//				Log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to unlock %d, %d\n", who, x, y)
//				//	WSYXLocks[y][x].Unlock()
//			}
//		}
//	}
//}

//func unlockAllYXs(who string) {
//	unlockYXRangeInclusive(0, GRID_HEIGHT-1, 0, GRID_WIDTH-1, who)
//}
