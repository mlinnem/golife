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

const RANDOM_SEED = false

const MAX_WSS = 19000

//---PERFORMANCE_VARIABLES---
//--GENERAL---

//NOTE: If this is on, CellActionDecider can't be parallel
const SERIAL_ACTIONTOEXECUTION_BRIDGE = false
const DEFAULT_BRIDGE_ACTIONCAPACITY = MAX_CELL_COUNT

const PERSISTENT_CELLACTIONDECIDER = true
const PARALLELIZE_DECIDER = false
const CELLACTIONDECIDER_ROUTINECOUNT = 5
const CELLS_PER_CELLDECIDERBUNDLE = 1000

const PERSISTENT_CELLACTIONEXECUTER = false
const PARALLELIZE_CELLEXECUTER = false //Probably not good to say true for thread-safety reasons
const CELLACTIONEXECUTER_ROUTINECOUNT = 1

const PERSISTENT_NONCELLACTIONEXECUTER = false
const PARALLELIZE_NONCELLEXECUTER = false //Probably not good to say true for thread-safety reasons
const NONCELLACTIONEXECUTER_ROUTINECOUNT = 1

//--SPECIFIC--
const PARALLELIZE_SHINE_BY_ROW = true

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
		TracedCell = WS.Cells[0]
	}
}

func generateSystemNonCellActions() {
	//generate any nonCellActions that need to be generated
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

	for WSNum = 0; WSNum < MAX_WSS; WSNum++ {
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
	cellaction_growcanopy       = iota
	cellaction_growheight       = iota
	cellaction_growlegs         = iota
	cellaction_growchloroplasts = iota
	cellaction_wait             = iota
	cellaction_reproduce        = iota
	cellaction_moverandom       = iota
	specialcellaction_done      = iota
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
	case cellaction_growlegs:
		return "Grow Legs"
	case cellaction_moverandom:
		return "Move Random"
	case cellaction_growchloroplasts:
		return "Grow Chloroplasts"
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
				LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: sending action '%s'\n", cell.ID, getCellActionName(cellAction.actionType))
				//TODO: Is this safe when executer is persistent?
				if SERIAL_ACTIONTOEXECUTION_BRIDGE {
					serialQueuedCellActions[serialQueuedCellActionsLength] = cellAction
					serialQueuedCellActionsLength++
				} else {
					queuedCellActions <- cellAction
				}
			} else {
				LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: NO ACTION\n", cell.ID)
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
	} else if cell.IsReadyToGrowCanopy() {
		cellAction = &CellAction{cell, cellaction_growcanopy}
	} else if cell.IsReadyToGrowLegs() {
		cellAction = &CellAction{cell, cellaction_growlegs}
	} else if cell.IsReadyToGrowChloroplasts() {
		cellAction = &CellAction{cell, cellaction_growchloroplasts}
	} else if cell.IsTimeToReproduce() {
		cellAction = &CellAction{cell, cellaction_reproduce}
	} else if cell.WantsToAndCanMove() {
		cellAction = &CellAction{cell, cellaction_moverandom}
	} else if cell.ShouldWait() {
		cellAction = &CellAction{cell, cellaction_wait}
	} else {
		//no action at all. Hopefully don't need to submit a null action
	}

	LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: PAY THINKING\n", cell.ID)
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
	case cellaction_growcanopy:
		cell.GrowCanopy()
	case cellaction_growheight:
		cell.GrowHeight()
	case cellaction_growchloroplasts:
		cell.GrowChloroplasts()
	case cellaction_wait:
		cell.Wait()
	case cellaction_growlegs:
		cell.GrowLegs()
	case cellaction_moverandom:
		cell.MoveRandom()
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

var HIGH_MUTATION_CHANCE = 3
var NO_MUTATION_CHANCE = 50

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
	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y
		if !WS.IsSolidOrCovered(xTry, yTry, z) {
			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Making baby from %d, %d -> %d, %d\n", cell.ID, cell.X, cell.Y, xTry, yTry)
			var babyCell = CellPool.Borrow()
			babyCell.Energy = cell.EnergySpentOnReproducing - REPRODUCE_COST
			//TODO: This should be a function, probably needs locking if parallelized
			babyCell.ID = IDCounter
			IDCounter++
			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d born with %f energy\n", babyCell.ID, babyCell.Energy)
			babyCell.X = xTry
			babyCell.Y = yTry
			babyCell.Z = cell.Z //TODO: Might want to revisit this later?
			babyCell.SpeciesID = cell.SpeciesID
			babyCell.SpeciesColor = cell.SpeciesColor
			babyCell.TimeLeftToWait = 0
			babyCell.Age = 0
			babyCell.Canopy = false
			babyCell.Height = 0
			babyCell.Legs = false
			babyCell.Chloroplasts = false

			//var rand0To99 = rand.Intn(100)
			//if rand0To99 < 10 {
			//	babyCell.ClockRate = int(math.Max(1.0, float64(cell.ClockRate+rand.Intn(17)-8)))
			//} else if rand0To99 < 80 {
			//	babyCell.ClockRate = int(math.Max(1.0, float64(cell.ClockRate+rand.Intn(3)-1)))
			//} else {
			//	babyCell.ClockRate = cell.ClockRate
			//}
			//TODO: Better way to do this

			babyCell.ClockRate = int(mutateMinMaxInThisRange(float64(cell.ClockRate), 8.0, 50.0, 1.0, 1000.0))
			babyCell.EnergySpentOnReproducing = mutateMinMaxInThisRange(cell.EnergySpentOnReproducing, 8.0, 1000.0, REPRODUCE_COST, 2000.0)
			babyCell.EnergyReproduceThreshold = mutateMinMaxInThisRange(cell.EnergyReproduceThreshold, 8.0, 1000.0, babyCell.EnergySpentOnReproducing, 2000.0)
			babyCell.GrowChloroplastsAt = mutateMinMaxInThisRange(cell.GrowChloroplastsAt, 8.0, 1000.0, GROWCHLOROPLASTS_COST, 1000.0)
			babyCell.GrowCanopyAt = mutateMinMaxInThisRange(cell.GrowCanopyAt, 8.0, 1000.0, GROWCANOPY_COST, 2000.0)
			babyCell.GrowHeightAt = mutateMinMaxInThisRange(cell.GrowHeightAt, 8.0, 1000.0, GROWHEIGHT_COST, 2000.0) //math.Max(GROWHEIGHT_COST, cell.GrowHeightAt+float64(rand.Intn(9)-5))
			babyCell.GrowLegsAt = mutateMinMaxInThisRange(cell.GrowLegsAt, 8.0, 1000.0, GROWLEGS_COST, 2000.0)
			babyCell.MoveChance = mutateMinMaxInThisRange(cell.MoveChance, 6.0, 100.0, 0, 100.0)
			babyCell.PercentChanceWait = int(mutateMinMaxInThisRange(float64(cell.PercentChanceWait), 6.0, 100.0, 0, 100.0))

			//var rand0To99_2 = rand.Intn(100)
			//if rand0To99_2 == 0 {
			//	//legless gets legs
			//	babyCell.GrowLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST
			//	babyCell.MoveChance = float64(rand.Intn(40))
			//} else if rand0To99_2 == 1 {
			//legged goes legless
			//	babyCell.GrowLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST + 1000
			//	babyCell.MoveChance = 0.0
			//} else {
			//mutate like normal
			//	babyCell.GrowLegsAt = math.Max(GROWLEGS_COST, float64(cell.GrowLegsAt+float64(rand.Intn(7)-3)))
			//	babyCell.MoveChance = math.Max(0.0, float64(cell.MoveChance+float64(rand.Intn(7)-3)))
			//}

			//babyCell.PercentChanceWait = int(math.Max(0.0, float64(cell.PercentChanceWait+rand.Intn(7)-3)))
			babyCell.X_originalGrowChloroplastsAt = cell.X_originalGrowChloroplastsAt
			babyCell.X_originalClockRate = cell.X_originalClockRate
			babyCell.X_originalEnergyReproduceThreshold = cell.X_originalEnergyReproduceThreshold
			babyCell.X_originalEnergySpentOnReproducing = cell.X_originalEnergySpentOnReproducing
			babyCell.X_originalGrowCanopyAt = cell.X_originalGrowCanopyAt
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

//func lockAllYXs(who string) {
//	lockYXRangeInclusive(0, GRID_HEIGHT-1, 0, GRID_WIDTH-1, who)
//}

//func lockYXRangeInclusive(startY int, endY int, startX int, endX int, who string) {
//	Log(LOGTYPE_DEBUGCONCURRENCY, "%s Trying to grab bulk lock\n", who)
//	bulkGrabLock.Lock()
//	Log(LOGTYPE_DEBUGCONCURRENCY, "%s Grabbed successfully\n", who)
//	for y := startY; y < endY+1; y++ {
//		for x := startX; x < endX+1; x++ {
//			if !WS.IsOutOfBounds(x, y, z) {
//				Log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to lock %d, %d\n", who, x, y)
//				//WSYXLocks[y][x].Lock()
//			}
//		}
//	}
//	Log(LOGTYPE_DEBUGCONCURRENCY, "%s trying to release bulk lock\n", who)
//	bulkGrabLock.Unlock()
//}

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
			newCell.Energy = float64(rand.Intn(100))
			newCell.TimeLeftToWait = 0
			newCell.Chloroplasts = false

			newCell.ClockRate = rand.Intn(400) + 1
			newCell.PercentChanceWait = rand.Intn(90)

			newCell.EnergySpentOnReproducing = REPRODUCE_COST + float64(rand.Intn(1500))
			newCell.EnergyReproduceThreshold = newCell.EnergySpentOnReproducing + float64(rand.Intn(1500))
			newCell.Canopy = false
			newCell.GrowChloroplastsAt = float64(rand.Intn(20)) + GROWCHLOROPLASTS_COST //float64(rand.Intn(1500)) + GROWCHLOROPLASTS_COST
			newCell.GrowCanopyAt = float64(rand.Intn(1500)) + GROWCANOPY_COST
			newCell.GrowHeightAt = float64(rand.Intn(1500)) + GROWHEIGHT_COST
			if rand.Intn(100) < 50 {
				newCell.GrowLegsAt = float64(rand.Intn(1500)) + GROWLEGS_COST
				newCell.MoveChance = float64(rand.Intn(90))
			} else {
				newCell.GrowLegsAt = float64(rand.Intn(1500)) + GROWLEGS_COST
				newCell.MoveChance = 0.0
			}

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

const SHINE_FREQUENCY = 5

func shineOnAllCells() {
	if WSNum%SHINE_FREQUENCY == 0 {
		Log(LOGTYPE_MAINLOOPSINGLE, "Starting shine\n")
		//TODO: This could stand to be refactored a bit
		//TODO: Changed for WS to current WS, but might be wrong...

		var isDayTime = WSNum%100 <= 50
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

func shineThisRow(yi int, isDayTime bool, wg *sync.WaitGroup) {
	for xi := 0; xi < GRID_WIDTH; xi++ {
		if isDayTime { //int(YProximityToMiddleAsPercent(yi)*100)

			var shineAmountForThisSquare float64
			if xi%2 == 0 && yi%2 == 0 {
				shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * SHINE_FREQUENCY * 2 * 2 //* float64(float64(yi)/float64(GRID_HEIGHT))

			} else {
				shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * SHINE_FREQUENCY * (float64(xi) / GRID_WIDTH) * 2 * 2 //* float64(float64(yi)/float64(GRID_HEIGHT))
			}
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

func newShineMethod(x int, y int, shineAmountForThisSquare float64) {

	//TODO: Just making the array is faster than using pool. Surprising
	var surroundingCellsWithCanopiesAndMe = &[SURROUNDINGS_SIZE]*Cell{} //surroundingsPool.Borrow()
	var numSurrounders = 0
	//TODO: Need to rejigger to take into account 3rd dimension better
	var z = 0

	var cell = WS.SpatialIndexSurfaceCover[z][y][x]

	//If we have a cell that is tall...
	if cell != nil && cell.Chloroplasts == true && cell.Height == 1 {
		//give all energy to that cell
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: Shine @ height 1\n", cell.ID)

		cell.IncreaseEnergy(shineAmountForThisSquare)
	} else {
		//otherwise, distribute energy among the cell (if it exists), and surrounding cells that have canopies
		if cell != nil && cell.Chloroplasts == true {
			surroundingCellsWithCanopiesAndMe[0] = cell
			numSurrounders++
		}
		for relativeY := -1; relativeY < 2; relativeY++ {
			for relativeX := -1; relativeX < 2; relativeX++ {
				var xTry = x + relativeX
				var yTry = y + relativeY
				if relativeX == 0 && relativeY == 0 {
					continue
				}
				if !WS.IsOutOfBounds(xTry, yTry, z) && WS.IsSolidOrCovered(xTry, yTry, z) && WS.SpatialIndexSurfaceCover[z][yTry][xTry].Canopy == true {
					var surroundingCell = WS.SpatialIndexSurfaceCover[z][yTry][xTry]
					surroundingCellsWithCanopiesAndMe[numSurrounders] = surroundingCell
					numSurrounders++
				}
			}
		}
		var energyToEachCell = shineAmountForThisSquare / float64(numSurrounders)
		for i := 0; i < numSurrounders; i++ {
			LogIfTraced(surroundingCellsWithCanopiesAndMe[i], LOGTYPE_CELLEFFECT, "cell %d: Shine @ height 0, 1/%d\n", surroundingCellsWithCanopiesAndMe[i].ID, numSurrounders)
			surroundingCellsWithCanopiesAndMe[i].IncreaseEnergy(energyToEachCell)
		}
	}
}

func newerShineMethod(x int, y int, shineAmountForThisSquare float64) {
	//	fmt.Printf("Shinin \n")

	var shineAmountLeft = shineAmountForThisSquare
	for z := GRID_DEPTH - 1; z >= 0; z-- {
		//	fmt.Printf("Blam")
		var remaining = giveEnergyToNonSolidCellsAtThisLevel(x, y, z, shineAmountLeft/2)
		//TODO: This needs to handle roofs
		shineAmountLeft = shineAmountLeft/2 + remaining
		if z == 0 {
			var cell = WS.SpatialIndexSurfaceCover[z][y][x]
			if cell != nil && cell.Chloroplasts == true {
				cell.IncreaseEnergy(shineAmountLeft)
			}
			//We're about to hit ground cover
		} else {
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
		LogIfTraced(surroundingCellsWithCanopiesAndMe[i], LOGTYPE_CELLEFFECT, "cell %d: Shine @ height 0, 1/%d\n", surroundingCellsWithCanopiesAndMe[i].ID, numSurrounders)
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
