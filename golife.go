package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/davecheney/profile"
	. "github.com/mlinnem/golife/main/lib"
)

var bulkGrabLock *sync.Mutex

var cellActionExecuterWG sync.WaitGroup

var cellsReadyForAction = make(chan []*Cell, MAX_CELL_COUNT)

var queuedNonCellActions []*NonCellAction
var pendingNonCellActions = make(chan *NonCellAction, MAX_CELL_COUNT)

var pendingCellActions = make(chan *CellAction, MAX_CELL_COUNT)

var cellActionDeciderWG sync.WaitGroup
var nonCellActionExecuterWG sync.WaitGroup

var waitForCleaning sync.WaitGroup

var momentNum = 0

func startPersistentThreads() {
	//set up nonCellActionExecutors

	for i := 0; i < NONCELLACTIONDECIDER_ROUTINECOUNT; i++ {
		go nonCellActionExecuter(&nonCellActionExecuterWG)
	}
	//set up cellActionDeciders to pull from readyCells channel (freely)
	for i := 0; i < CELLACTIONDECIDER_ROUTINECOUNT; i++ {
		go cellActionDecider(&cellActionDeciderWG)
	}
	for i := 0; i < CELLACTIONEXECUTER_ROUTINECOUNT; i++ {
		go cellActionExecuter(&cellActionExecuterWG)
	}
}

func transferLiveCellsToNextMoment() {
	NextMoment.Cells = make([]*Cell, 0, len(CurrentMoment.Cells))
	NextMoment.CellsSpatialIndex = [GRID_WIDTH][GRID_HEIGHT]*Cell{}
	for _, CurrentMomentCell := range CurrentMoment.Cells {
		//This takes place of reaper function
		//Log(LOGTYPE_HIGHFREQUENCY, "a cell %d, has %6.2f energy\n", CurrentMomentCell._secretID, CurrentMomentCell.Energy)
		if CurrentMomentCell.Energy > 0 {
			var NextMomentCell = CurrentMomentCell.ContinueOn()
			NextMoment.Cells = append(NextMoment.Cells, NextMomentCell)
			NextMoment.CellsSpatialIndex[NextMomentCell.X][NextMomentCell.Y] = NextMomentCell
		}
	}
	Log(LOGTYPE_MAINLOOPSINGLE, "Transferred cells over to next moment by default, same loc\n")
}

func feedCurrentMomentCellsToActionDecider() {
	var startSlice = 0
	var endSlice = CELLS_PER_BUNDLE
	for {
		if endSlice < len(CurrentMoment.Cells) {
			cellActionDeciderWG.Add(1)
			cellsReadyForAction <- CurrentMoment.Cells[startSlice:endSlice]
			startSlice += CELLS_PER_BUNDLE
			endSlice += CELLS_PER_BUNDLE
		} else {
			cellActionDeciderWG.Add(1)
			cellsReadyForAction <- CurrentMoment.Cells[startSlice:]
			break
		}
	}
}

func feedQueuedNonCellActionsToExecuter() {
	for ai := 0; ai < len(queuedNonCellActions); ai++ {
		var nonCellAction = queuedNonCellActions[ai]
		pendingNonCellActions <- nonCellAction
	}
	//empty queue now that they've been sent. Probably better way to do this
	queuedNonCellActions = []*NonCellAction{}
}

func main() {

	//FIRST-TIME INIT
	if RANDOM_SEED {
		rand.Seed(int64(time.Now().Second()))
	}
	defer profile.Start(profile.CPUProfile).Stop()

	CurrentMoment = &Moment{}
	NextMoment = &Moment{}
	MomentBeingCleaned = &Moment{}

	bulkGrabLock = &sync.Mutex{}

	startPersistentThreads()

	var t1all = time.Now()
	for momentNum = 0; momentNum < MAX_MOMENTS; momentNum++ {
		CurrentMoment.MomentNum = momentNum
		NextMoment.MomentNum = momentNum + 1
		if momentNum%PRINTGRID_EVERY_N_TURNS == 0 {
			PrintGrid(CurrentMoment)
			PrintSpeciesReport(CurrentMoment, NUM_TOP_SPECIES_TO_PRINT)
		}
		var t1a = time.Now()
		//var t1 = time.Now()
		Log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "moment %d...\n", momentNum)
		//Assume all cells will be in same position in next moment
		//TODO: Should this be happening elsewhere?
		transferLiveCellsToNextMoment()

		feedCurrentMomentCellsToActionDecider()

		//NOT going to wait for all actions to be decided before executing. Decisions can trigger actions right away (prepare for locking needs...)
		//wait for all actionPickers and actionExecuters to be done  (NextMoment should be fully populated now)
		cellActionDeciderWG.Wait()
		cellActionExecuterWG.Wait()

		//generate any nonCellActions that need to be generated
		if momentNum == 0 {
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
		//feed waitingNonCellActions to readyNonCellActions
		Log(LOGTYPE_MAINLOOPSINGLE, "Dumping non-cell actions into working queue (if any)\n")

		feedQueuedNonCellActionsToExecuter()

		Log(LOGTYPE_MAINLOOPSINGLE, "Transferred to non-cell executers...\n")
		//wait for all non-cell actions to be done
		nonCellActionExecuterWG.Wait()

		Log(LOGTYPE_MAINLOOPSINGLE, "Non-cell Executers did their thing\n")
		//NextMoment becomes current moment.
		Log(LOGTYPE_MAINLOOPSINGLE, "Right before we switch to next moment")
		OldMoment = CurrentMoment
		CurrentMoment = NextMoment
		var tClean1 = time.Now()
		waitForCleaning.Wait()
		var tClean2 = time.Now()
		var durClean = tClean1.Sub(tClean2).Seconds()
		Log(LOGTYPE_MAINLOOPSINGLE, "Waiting on cleaning took this long: %d\n", durClean)
		var t2a = time.Now()
		NextMoment = MomentBeingCleaned
		MomentBeingCleaned = OldMoment
		Log(LOGTYPE_MAINLOOPSINGLE, "Right after the switcheroo")
		Log(LOGTYPE_MAINLOOPSINGLE, "About to wait for cleaning\n")
		waitForCleaning.Add(1)
		MomentBeingCleaned.ReturnCellsToPool()
		go MomentBeingCleaned.Clean(&waitForCleaning)

		var dura = t2a.Sub(t1a).Seconds()
		Log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "Time of entire turn took %f\n", dura)
		if len(CurrentMoment.Cells) == 0 {
			Log(LOGTYPE_FAILURES, "Early termination due to all cells dying\n")
			break
		}
	}

	Log(LOGTYPE_FINALSTATS, "%d cells in final moment\n", len(CurrentMoment.Cells))
	var t2all = time.Now()

	var durall = t2all.Sub(t1all).Seconds()
	Log(LOGTYPE_FINALSTATS, "Time of entire run took %f\n", durall)
}

//CELL-ACTION DECIDER

type CellAction struct {
	cell       *Cell
	actionType int
}

const (
	cellaction_reproduce  = iota
	cellaction_growcanopy = iota
	cellaction_wait       = iota
	cellaction_growlegs   = iota
	cellaction_moverandom = iota
)

func cellActionDecider(wg *sync.WaitGroup) {
	for {
		var cellSlice = <-cellsReadyForAction
		for _, cell := range cellSlice {
			if cell.TimeLeftToWait > 0 {
				cell.CountDown_TimeLeftToWait()
			} else {
				if cell.IsReadyToGrowCanopy() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_growcanopy}
				} else if cell.IsReadyToGrowLegs() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_growlegs}
				} else if cell.IsTimeToReproduce() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_reproduce}
				} else if cell.WantsToAndCanMove() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_moverandom}
				} else if cell.ShouldWait() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_wait}
				} else {
					//no action at all. Hopefully don't need to submit a null action
				}
				cell.DecreaseEnergy(THINKING_COST)
				cell.IncreaseWaitTime(cell.ClockRate - 1)
				//TODO: Make sure not to do off-by-one here. Clock 1 should be every 1 turn
			}
		}
		wg.Done()
	}
}

//CELL-ACTION EXECUTER

func cellActionExecuter(wg *sync.WaitGroup) {
	for {
		var cellAction = <-pendingCellActions
		switch cellAction.actionType {
		case cellaction_reproduce:
			reproduce(cellAction.cell)
		case cellaction_growcanopy:
			cellAction.cell.GrowCanopy()
		case cellaction_wait:
			cellAction.cell.Wait()
		case cellaction_growlegs:
			cellAction.cell.GrowLegs()
		}
		wg.Done()
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
		wg.Add(1)
		queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_spontaneouslyPlaceCell})
	}
}

func generateSunshineAction(wg *sync.WaitGroup) {
	//TODO for now this action is atom. Could break it up later.
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_shineOnAllCells})
	wg.Add(1)
}

func generateCellMaintenanceAction(wg *sync.WaitGroup) {
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_cellMaintenance})
	wg.Add(1)
}

func nonCellActionExecuter(wg *sync.WaitGroup) {
	for {
		var nonCellAction = <-pendingNonCellActions
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

		wg.Done()
	}
}

//TODO: Should be in cell.go?
func reproduce(cell *Cell) {
	if cell.Energy < REPRODUCE_COST || cell.Energy < cell.EnergySpentOnReproducing {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.DecreaseEnergy(cell.EnergySpentOnReproducing)
		return
	}
	lockAllYXs("reproduce")
	//try all spots surrounding the cell
	for _, direction := range GetSurroundingDirectionsInRandomOrder() {
		var xTry = cell.X + direction.X
		var yTry = cell.Y + direction.Y
		if !NextMoment.IsOccupied(xTry, yTry) {

			var rand0To99 = rand.Intn(100)
			Log(LOGTYPE_CELLEFFECT, "cell %d at %d, %d making a baby\n", cell.ID, cell.X, cell.Y)
			var babyCell = CellPool.Borrow()
			babyCell.Energy = cell.EnergySpentOnReproducing - REPRODUCE_COST
			//TODO: This should be a function, probably needs locking if parallelized
			babyCell.ID = IDCounter
			IDCounter++
			Log(LOGTYPE_CELLEFFECT, "baby cell %d born with %f energy\n", babyCell.ID, babyCell.Energy)
			babyCell.EnergyReproduceThreshold = cell.EnergyReproduceThreshold + float64(rand.Intn(7)-3)
			babyCell.X = xTry
			babyCell.Y = yTry
			babyCell.SpeciesID = cell.SpeciesID
			babyCell.SpeciesColor = cell.SpeciesColor

			babyCell.TimeLeftToWait = 0
			if rand0To99 < 40 {
				babyCell.ClockRate = cell.ClockRate + rand.Intn(9) - 4
			} else if rand0To99 < 80 {
				babyCell.ClockRate = cell.ClockRate + rand.Intn(3) - 1
			} else {
				babyCell.ClockRate = cell.ClockRate
			}
			//TODO: Better way to do this

			//TODO: This whole way of doing legs is weird. Should be on or off switch methinks
			//var rand0To99_2 = rand.Intn(100)
			//if rand0To99_2 == 0 {
			//legless gets legs
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
			//TODO: Just trying to disable this
			babyCell.GrowLegsAt = 999

			babyCell.PercentChanceWait = int(math.Max(0.0, float64(cell.PercentChanceWait+rand.Intn(7)-3)))

			babyCell.Age = 0
			babyCell.Canopy = false
			babyCell.GrowCanopyAt = cell.GrowCanopyAt + float64(rand.Intn(7)-3)

			babyCell.X_originalEnergyReproduceThreshold = cell.X_originalEnergyReproduceThreshold
			babyCell.X_originalEnergySpentOnReproducing = cell.X_originalEnergySpentOnReproducing
			babyCell.X_originalGrowCanopyAt = cell.X_originalGrowCanopyAt
			babyCell.X_originalPercentChanceWait = cell.X_originalPercentChanceWait
			babyCell.X_originalClockRate = cell.X_originalClockRate
			babyCell.X_originalGrowLegsAt = cell.X_originalGrowLegsAt
			babyCell.X_originalMoveChance = cell.X_originalMoveChance

			babyCell.EnergySpentOnReproducing = math.Min(babyCell.EnergyReproduceThreshold, cell.EnergySpentOnReproducing+float64(rand.Intn(7)-3))

			//TODO: WTF this shit is hella bugged in some mysterious way
			if hasSignificantGeneticDivergence(babyCell) {
				//TODO: This should probably be a function
				babyCell.SpeciesID = SpeciesIDCounter
				SpeciesIDCounter++
				babyCell.SpeciesColor = getNextColor()

				babyCell.X_originalEnergyReproduceThreshold = babyCell.EnergyReproduceThreshold
				babyCell.X_originalGrowCanopyAt = babyCell.GrowCanopyAt
				babyCell.X_originalEnergySpentOnReproducing = babyCell.EnergySpentOnReproducing
				babyCell.X_originalPercentChanceWait = babyCell.PercentChanceWait
				babyCell.X_originalClockRate = babyCell.ClockRate
				babyCell.X_originalGrowLegsAt = babyCell.GrowLegsAt
				babyCell.X_originalMoveChance = babyCell.MoveChance

				Log(LOGTYPE_SPECIALEVENT, "Cell at %d, %d "+babyCell.SpeciesColor.StartSequence+"x"+babyCell.SpeciesColor.EndSequence+" is the first of a new species!\n", xTry, yTry)
				Log(LOGTYPE_SPECIALEVENT, "orig reprod threshold: %f, new reprod threshold: %f\n", cell.X_originalEnergyReproduceThreshold, babyCell.X_originalEnergyReproduceThreshold)
				Log(LOGTYPE_SPECIALEVENT, "orig reprod energy spend: %f, new reprod energy spend: %f\n", cell.X_originalEnergySpentOnReproducing, babyCell.X_originalEnergySpentOnReproducing)
				Log(LOGTYPE_SPECIALEVENT, "orig grow canopy threshold: %f, new grow canopy threshold: %f\n", cell.X_originalGrowCanopyAt, babyCell.X_originalGrowCanopyAt)

			}
			NextMoment.Cells = append(NextMoment.Cells, babyCell)
			NextMoment.CellsSpatialIndex[xTry][yTry] = babyCell
			cell.DecreaseEnergy(cell.EnergySpentOnReproducing)
			return
		}
	}
	//	unlockYRangeInclusive(cell.Y-1, cell.Y+1, "reproduce")
}

func hasSignificantGeneticDivergence(cell *Cell) bool {
	var energyReproduceThresholdDiff = math.Abs(cell.X_originalEnergyReproduceThreshold - cell.EnergyReproduceThreshold)
	//TODO: took canopy out because something is jacked up about it. Need to debug and put back in
	//var GrowCanopyAtDiff = 0.0
	//if cell.GrowCanopyAt != 0.0 {
	//	GrowCanopyAtDiff = math.Abs(cell.X_OriginalGrowCanopyAt - cell.GrowCanopyAt)
	//}

	//var MoveChanceDiff = math.Abs(float64(cell._X_originalMoveChance) - float64(cell.MoveChance))
	//var GrowLegsAtDiff = math.Abs(float64(cell._X_originalGrowLegsAt) - float64(cell.GrowCanopyAt))
	var GrowCanopyAtDiff = math.Abs(float64(cell.X_originalGrowCanopyAt) - float64(cell.GrowCanopyAt))
	var ClockRateDiff = math.Abs(float64(cell.X_originalClockRate) - float64(cell.ClockRate))
	var EnergySpentOnReproducingDiff = math.Abs(cell.X_originalEnergySpentOnReproducing - cell.EnergySpentOnReproducing)
	var PercentChanceWaitDiff = math.Abs(float64(cell.X_originalPercentChanceWait) - float64(cell.PercentChanceWait))
	var totalDiff = GrowCanopyAtDiff + ClockRateDiff + energyReproduceThresholdDiff + EnergySpentOnReproducingDiff + PercentChanceWaitDiff
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

func maintainAllCells() {
	Log(LOGTYPE_MAINLOOPSINGLE, "Starting maintain\n")
	for _, cell := range CurrentMoment.Cells {
		cell.Maintain()
	}
	Log(LOGTYPE_MAINLOOPSINGLE, "Ending maintain\n")
}

func lockAllYXs(who string) {
	lockYXRangeInclusive(0, GRID_HEIGHT-1, 0, GRID_WIDTH-1, who)
}

func lockYXRangeInclusive(startY int, endY int, startX int, endX int, who string) {
	Log(LOGTYPE_DEBUGCONCURRENCY, "%s Trying to grab bulk lock\n", who)
	bulkGrabLock.Lock()
	Log(LOGTYPE_DEBUGCONCURRENCY, "%s Grabbed successfully\n", who)
	for y := startY; y < endY+1; y++ {
		for x := startX; x < endX+1; x++ {
			if !NextMoment.IsOutOfBounds(x, y) {
				Log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to lock %d, %d\n", who, x, y)
				//NextMomentYXLocks[y][x].Lock()
			}
		}
	}
	Log(LOGTYPE_DEBUGCONCURRENCY, "%s trying to release bulk lock\n", who)
	bulkGrabLock.Unlock()
}

func unlockYXRangeInclusive(startY int, endY int, startX int, endX int, who string) {
	for y := startY; y < endY+1; y++ {
		for x := startX; x < endX+1; x++ {
			if !NextMoment.IsOutOfBounds(x, y) {
				Log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to unlock %d, %d\n", who, x, y)
				//	NextMomentYXLocks[y][x].Unlock()
			}
		}
	}
}

func unlockAllYXs(who string) {
	unlockYXRangeInclusive(0, GRID_HEIGHT-1, 0, GRID_WIDTH-1, who)
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
	for !foundSpotYet && !giveUp {

		var xTry = rand.Intn(GRID_WIDTH)
		var yTry = rand.Intn(GRID_HEIGHT)
		if !NextMoment.IsOccupied(xTry, yTry) {
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

			newCell.ClockRate = rand.Intn(50) + 1
			newCell.PercentChanceWait = rand.Intn(40)

			newCell.EnergySpentOnReproducing = REPRODUCE_COST + float64(rand.Intn(20))
			newCell.EnergyReproduceThreshold = newCell.EnergySpentOnReproducing + float64(rand.Intn(120))
			newCell.Canopy = false
			newCell.GrowCanopyAt = float64(rand.Intn(120)) + GROWCANOPY_COST
			//if rand.Intn(100) < 50 {
			//	newCell.GrowLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST
			//	newCell.MoveChance = float64(rand.Intn(40))
			//} else {
			//		newCell.GrowLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST + 1000
			//		newCell.MoveChance = 0.0
			//	}
			newCell.GrowLegsAt = 9999
			newCell.Legs = false

			newCell.X_originalMoveChance = newCell.MoveChance
			newCell.X_originalGrowLegsAt = newCell.GrowLegsAt
			newCell.X_originalPercentChanceWait = newCell.PercentChanceWait
			newCell.X_originalEnergyReproduceThreshold = newCell.EnergyReproduceThreshold
			newCell.X_originalEnergySpentOnReproducing = newCell.EnergySpentOnReproducing
			newCell.X_originalClockRate = newCell.ClockRate
			newCell.X_originalGrowCanopyAt = newCell.GrowCanopyAt

			NextMoment.Cells = append(NextMoment.Cells, newCell)
			Log(LOGTYPE_CELLEFFECT, "Added cell %d to next moment\n", newCell.ID)
			NextMoment.CellsSpatialIndex[xTry][yTry] = newCell
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

const SHINE_FREQUENCY = 1

func shineOnAllCells() {
	if momentNum%SHINE_FREQUENCY == 0 {
		Log(LOGTYPE_MAINLOOPSINGLE, "Starting shine\n")
		//TODO: This could stand to be refactored a bit
		//TODO: Changed for NextMoment to current moment, but might be wrong...

		var isDayTime = momentNum%100 <= 50
		var wg = &sync.WaitGroup{}
		for yi := range CurrentMoment.CellsSpatialIndex {
			//wg.Add(1)
			shineThisRow(yi, isDayTime, wg)

			//Log(LOGTYPE_HIGHFREQUENCY, "Shiner touching on %d, %d \n", xi, yi)
			//unlockYXRangeInclusive(yi-1, yi+1, xi-1, xi+1, "shiner")
			//TODO: May have placed lock/unlocks here incorrectly
		}
		Log(LOGTYPE_MAINLOOPSINGLE, "Ending shine\n")
		//wg.Wait()
	}

}

func shineThisRow(yi int, isDayTime bool, wg *sync.WaitGroup) {
	for xi := range CurrentMoment.CellsSpatialIndex[yi] {
		//var cell = CurrentMoment.CellsSpatialIndex[yi][xi]
		//if cell == nil {
		//	continue
		//}
		//	lockYXRangeInclusive(yi-1, yi+1, xi-1, xi+1, "shiner")
		//fmt.Printf("Proximity to middle: %d\n", int(YProximityToMiddleAsPercent(yi)*100))
		if isDayTime { //int(YProximityToMiddleAsPercent(yi)*100)

			//var shineAmountForThisSquare = SHINEEnergy_AMOUNT //* (float64(xi) / float64(GRID_HEIGHT)) //  GRADIENT
			var shineAmountForThisSquare = SHINEEnergy_AMOUNT * SHINE_FREQUENCY * (float64(xi) / GRID_WIDTH) //* float64(float64(yi)/float64(GRID_HEIGHT))
			//fmt.Printf("shine amount at %d: %f", yi, shineAmountForThisSquare)
			//var shineAmountForThisSquare = 0.0
			//if xi%10 == 0 || (xi+1)%10 == 0 || (yi%10 == 0) || ((yi+1)%10) == 0 {
			//	shineAmountForThisSquare = 0.0
			//} else {
			//	shineAmountForThisSquare = SHINEEnergy_AMOUNT
			//}
			newShineMethod(xi, yi, shineAmountForThisSquare)
			//oldShineMethod(cell, shineAmountForThisSquare)
		} else {
			//No sun at night
		}
	}
	//wg.Done()
}

const SURROUNDINGS_SIZE = 9

func newShineMethod(x int, y int, shineAmountForThisSquare float64) {
	//TODO: Just making the array is faster than using pool. Surprising
	var surroundingCellsWithCanopiesAndMe = &[SURROUNDINGS_SIZE]*Cell{} //surroundingsPool.Borrow()
	var numSurrounders = 0
	var cell = CurrentMoment.CellsSpatialIndex[y][x]
	if cell != nil {
		surroundingCellsWithCanopiesAndMe[0] = cell
		numSurrounders++
	}
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = x + relativeX
			var yTry = y + relativeY
			if relativeX == 0 && relativeY == 0 {
				continue
			}
			if !CurrentMoment.IsOutOfBounds(xTry, yTry) && CurrentMoment.IsOccupied(xTry, yTry) && CurrentMoment.CellsSpatialIndex[xTry][yTry].Canopy == true {
				var surroundingCell = CurrentMoment.CellsSpatialIndex[xTry][yTry]
				surroundingCellsWithCanopiesAndMe[numSurrounders] = surroundingCell
				numSurrounders++
			}
		}
	}
	var energyToEachCell = shineAmountForThisSquare / float64(numSurrounders)
	for i := 0; i < numSurrounders; i++ {
		surroundingCellsWithCanopiesAndMe[i].IncreaseEnergy(energyToEachCell)
	}
}
