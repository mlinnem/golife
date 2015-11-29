package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/davecheney/profile"
)

var secretIDCounter = 0

func newCell() *Cell {
	secretIDCounter++
	return &Cell{_secretID: secretIDCounter}
}

type Pool struct {
	pool chan *Cell
}

// NewPool creates a new pool of Cells
func newPool(max int) *Pool {
	return &Pool{
		pool: make(chan *Cell, max),
	}
}

// Borrow a Cell from the pool.
func (p *Pool) Borrow() *Cell {
	var c *Cell
	select {
	case c = <-p.pool:
	default:
		c = newCell()
	}
	return c
}

// Return returns a Cell to the pool.
func (p *Pool) Return(c *Cell) {
	select {
	case p.pool <- c:
	default:

	}
}

//TODO: Make it easy to add a field and have it appear in all the right places Re: copying and whatnot
type Cell struct {
	energy         float64
	x              int
	y              int
	timeLeftToWait int
	canopy         bool
	grewCanopyAt   float64
	//TODO: Might be better way to not have to pass in the cell itself
	isReadyToGrowCanopy               func(*Cell) bool
	energySpentOnReproducing          float64
	nextMomentSelf                    *Cell
	age                               int
	percentChanceWait                 int //out of 100
	_secretID                         int
	_originalGrewCanopyAt             float64
	_originalPercentChanceWait        int
	_originalEnergySpentOnReproducing float64
	_originalEnergyReproduceThreshold float64
	energyReproduceThreshold          float64
	speciesColor                      *TextColorBookend
}

func (oldCell *Cell) Copy() *Cell {
	//TODO: Can this not be done by just a generic struct copy?
	var newCell = cellPool.Borrow()
	newCell.energy = oldCell.energy
	newCell.age = oldCell.age
	newCell.x = oldCell.x
	newCell.y = oldCell.y
	newCell.percentChanceWait = oldCell.percentChanceWait
	newCell.speciesColor = oldCell.speciesColor
	newCell.energySpentOnReproducing = oldCell.energySpentOnReproducing
	newCell.canopy = oldCell.canopy
	newCell.grewCanopyAt = oldCell.grewCanopyAt
	newCell.isReadyToGrowCanopy = oldCell.isReadyToGrowCanopy
	newCell.energyReproduceThreshold = oldCell.energyReproduceThreshold

	newCell._originalEnergyReproduceThreshold = oldCell._originalEnergyReproduceThreshold
	newCell._originalEnergySpentOnReproducing = oldCell._originalEnergySpentOnReproducing
	newCell._originalGrewCanopyAt = oldCell._originalGrewCanopyAt
	newCell._originalPercentChanceWait = oldCell._originalPercentChanceWait

	return newCell
}

func (oldCell *Cell) ContinueOn() *Cell {
	var continuedCell = oldCell.Copy()
	oldCell.nextMomentSelf = continuedCell
	return continuedCell
}

const CELL_LIFESPAN = 300

func (cell *Cell) Maintain() {
	cell.energy -= float64(cell.age) / CELL_LIFESPAN
	cell.age += 1
}

type CellAction struct {
	cell       *Cell
	actionType int
}

type NonCellAction struct {
	actionType int
}

const GRID_WIDTH = 50
const GRID_HEIGHT = 150

type Moment struct {
	cells             []*Cell
	cellsSpatialIndex [GRID_WIDTH][GRID_HEIGHT]*Cell
}

func (moment *Moment) RemoveCells(cellsToDelete []*Cell) {
	//TODO: Wow I bet this performance sucks. Can we do better?

	//w := 0 // write index
	data := []*Cell{}
loop:
	for _, cellThatExists := range moment.cells {
		for _, cellToDelete := range cellsToDelete {
			if cellThatExists == cellToDelete {
				cellPool.Return(cellToDelete)
				moment.cellsSpatialIndex[cellToDelete.x][cellToDelete.y] = nil
				continue loop
			}
		}
		//TODO: No way in hell does this have good performance
		data = append(data, cellThatExists)
		//w++
	}
	moment.cells = data

}

func (moment *Moment) ReturnCellsToPool() {
	for ci := range moment.cells {
		var cellToReturn = moment.cells[ci]
		cellPool.Return(cellToReturn)
	}
}

//TODO: Maybe do this in tandem with goroutine in future
func (moment *Moment) Clean(wg *sync.WaitGroup) {
	//nextMomentLock.Lock()
	//for ci := range moment.cells {
	//moment.cells[ci] = nil
	//}
	moment.cells = []*Cell{}
	//moment.cellsSpatialIndex = [][]*Cell{{}}
	//var internalwg sync.WaitGroup
	log(LOGTYPE_MAINLOOPSINGLE, "You made it to right before allocating the big cell thing\n")
	//TODO: Letting garbage collector take care of cleaning rather than manual for now
	moment.cellsSpatialIndex = [GRID_WIDTH][GRID_HEIGHT]*Cell{}
	// for yi := range moment.cellsSpatialIndex {
	// 	//internalwg.Add(1)
	// 	moment.CleanRow(yi)
	// }
	// //internalwg.Wait()
	wg.Done()
	//nextMomentLock.Unlock()
}

func (moment *Moment) CleanRow(yi int) {
	for xi := range moment.cellsSpatialIndex[yi] {
		moment.cellsSpatialIndex[xi][yi] = nil
	}
	//wg.Done()
}

var cellPool *Pool

var oldMoment *Moment
var momentBeingCleaned *Moment
var momentReadyToBeNext *Moment
var currentMoment *Moment
var nextMoment *Moment
var nextMomentLock *sync.Mutex
var nextMomentYLocks [GRID_HEIGHT]sync.Mutex
var bulkGrabLock *sync.Mutex

const initialCellCount = 100
const maxCellCount = 9000000

//20.7 secs at 10,10,10
//20.6 at 10, 100, 10
//20.9 at 10, 1, 10
//21.3 at 10, 1000, 10
//21.2 at 10, 50, 10
//21,5 at 10, 400, 10
const cellActionDeciderRoutineCount = 1
const cellActionExecuterRoutineCount = 2

var nonCellActionExecuterRoutineCount = int(math.Max(1, (GRID_WIDTH*GRID_HEIGHT)/20))

var cellActionExecuterWG sync.WaitGroup

const MAX_MOMENTS = 49000

var cellsReadyForAction = make(chan *Cell, maxCellCount)

var queuedNonCellActions []*NonCellAction
var pendingNonCellActions = make(chan *NonCellAction, maxCellCount)
var pendingCellActions = make(chan *CellAction, maxCellCount)

func createThisManyCells(startHere int, endBeforeHere int, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := startHere; i < endBeforeHere; i++ {
		cellPool.Borrow()
	}
}

var momentNum = 0

func main() {
	rand.Seed(int64(time.Now().Second()))
	defer profile.Start(profile.CPUProfile).Stop()
	nextMomentYLocks = [GRID_HEIGHT]sync.Mutex{}
	bulkGrabLock = &sync.Mutex{}
	cellPool = newPool(maxCellCount)

	//set up nonCellActionExecutors
	var nonCellActionExecuterWG sync.WaitGroup
	for i := 0; i < nonCellActionExecuterRoutineCount; i++ {
		go nonCellActionExecuter(&nonCellActionExecuterWG)
	}

	//set up cellActionDeciders to pull from readyCells channel (freely)
	var cellActionDeciderWG sync.WaitGroup
	for i := 0; i < cellActionDeciderRoutineCount; i++ {
		go cellActionDecider(&cellActionDeciderWG)
	}

	for i := 0; i < cellActionExecuterRoutineCount; i++ {
		go cellActionExecuter(&cellActionExecuterWG)
	}
	//set up cellActionExecuters to pull from readyCellActions channel (freely)

	//set up nonCellActionExecutors to pull from readyNonCellActions channel (freely)

	//request initial cells to be created

	currentMoment = &Moment{}
	nextMoment = &Moment{}
	momentBeingCleaned = &Moment{}
	nextMomentLock = &sync.Mutex{}

	var waitForCleaning sync.WaitGroup

	var t1all = time.Now()
	for momentNum := 0; momentNum < MAX_MOMENTS; momentNum++ {
		var t1a = time.Now()
		//var t1 = time.Now()
		log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "moment %d...\n", momentNum)
		//Assume all cells will be in same position in next moment
		//TODO: This should be a function somewhere?
		//	nextMomentLock.Lock()
		for ci := range currentMoment.cells {
			var currentMomentCell = currentMoment.cells[ci]
			var nextMomentCell = currentMomentCell.ContinueOn()
			nextMoment.cells = append(nextMoment.cells, nextMomentCell)
			nextMoment.cellsSpatialIndex[nextMomentCell.x][nextMomentCell.y] = nextMomentCell
		}
		log(LOGTYPE_MAINLOOPSINGLE, "Transferred cells over to next moment by default, same loc\n")

		//TODO Make these pointers later?

		//feed all cells in currentMoment to cellsReadyForAction channel
		for ci := 0; ci < len(currentMoment.cells); ci++ {
			var cell = currentMoment.cells[ci]
			cellActionDeciderWG.Add(1)
			cellsReadyForAction <- cell

		}
		//NOT going to wait for all actions to be decided before executing. Decisions can trigger actions right away (prepare for locking needs...)
		//wait for all actionPickers and actionExecuters to be done  (nextMoment should be fully populated now)
		cellActionDeciderWG.Wait()
		cellActionExecuterWG.Wait()

		//generate any nonCellActions that need to be generated
		if momentNum == 0 {
			log(LOGTYPE_MAINLOOPSINGLE, "Populating initial cells\n")
			generateInitialNonCellActions(&nonCellActionExecuterWG)
			//	close(queuedNonCellActions)
		} else {
			generateCellMaintenanceAction(&nonCellActionExecuterWG)
			generateGrimReaperAction(&nonCellActionExecuterWG)
			generateSunshineAction(&nonCellActionExecuterWG)
		}
		//feed waitingNonCellActions to readyNonCellActions
		log(LOGTYPE_MAINLOOPSINGLE, "Dumping non-cell actions into working queue (if any)\n")

		for ai := 0; ai < len(queuedNonCellActions); ai++ {
			var nonCellAction = queuedNonCellActions[ai]
			pendingNonCellActions <- nonCellAction
		}
		//empty queue now that they've been sent. Probably better way to do this
		queuedNonCellActions = []*NonCellAction{}
		log(LOGTYPE_MAINLOOPSINGLE, "Transferred to non-cell executers...\n")
		//wait for all non-cell actions to be done
		nonCellActionExecuterWG.Wait()
		log(LOGTYPE_MAINLOOPSINGLE, "Non-cell Executers did their thing\n")

		//nextMoment becomes current moment.
		oldMoment = currentMoment
		currentMoment = nextMoment
		var tClean1 = time.Now()
		waitForCleaning.Wait()
		var tClean2 = time.Now()
		var durClean = tClean1.Sub(tClean2).Seconds()
		log(LOGTYPE_MAINLOOPSINGLE, "Waiting on cleaning took this long: %d\n", durClean)
		var t2a = time.Now()
		nextMoment = momentBeingCleaned
		momentBeingCleaned = oldMoment
		log(LOGTYPE_MAINLOOPSINGLE, "About to wait for cleaning\n")
		waitForCleaning.Add(1)
		momentBeingCleaned.ReturnCellsToPool()
		go momentBeingCleaned.Clean(&waitForCleaning)

		var dura = t2a.Sub(t1a).Seconds()
		log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "Time of entire turn took %f\n", dura)

		if momentNum%printGridEveryNTurns == 0 {
			printGrid(currentMoment)
		}
		if len(currentMoment.cells) == 0 {
			log(LOGTYPE_FAILURES, "Early termination due to all cells dying\n")
			break
		}
	}

	log(LOGTYPE_FINALSTATS, "%d cells in final moment\n", len(currentMoment.cells))
	var t2all = time.Now()

	var durall = t2all.Sub(t1all).Seconds()
	log(LOGTYPE_FINALSTATS, "Time of entire run took %f\n", durall)
}

func makeCanopyBuildConditionOnCertainEnergy(energyLevel float64) func(*Cell) bool {
	var makeCanopyEnergyThreshold = energyLevel
	var buildConditionFunction = func(cell *Cell) bool {
		return cell.canopy == false && cell.energy > makeCanopyEnergyThreshold
	}
	return buildConditionFunction
}

func printGrid(moment *Moment) {
	log(LOGTYPE_PRINTGRID_GRID, "\n")
	for row := range moment.cellsSpatialIndex {
		for col := range moment.cellsSpatialIndex[row] {
			var cell = moment.cellsSpatialIndex[row][col]
			if cell != nil {
				var colorStart = cell.speciesColor.startSequence
				var colorEnd = cell.speciesColor.endSequence
				if cell.canopy == true {
					log(LOGTYPE_PRINTGRID_GRID, colorStart+"X"+colorEnd)
					//log(LOGTYPE_PRINTGRID_GRID, "X")
				} else {
					log(LOGTYPE_PRINTGRID_GRID, colorStart+"x"+colorEnd)
					//log(LOGTYPE_PRINTGRID_GRID, "x")
				}
			} else {
				log(LOGTYPE_PRINTGRID_GRID, " ")
			}
		}
		log(LOGTYPE_PRINTGRID_GRID, "\n")
	}
	log(LOGTYPE_PRINTGRID_GRID, "\n")

	var energyReproduceThresholdTotal = 0.0
	var energySpentOnReproducingTotal = 0.0
	var canopyTotal = 0
	var grewCanopyAtTotal = 0.0
	var percentChanceWaitTotal = 0
	for ci := range moment.cells {
		var cell = moment.cells[ci]
		//TODO: Refactor all this crap to use cell variable
		energyReproduceThresholdTotal += moment.cells[ci].energyReproduceThreshold
		energySpentOnReproducingTotal += cell.energySpentOnReproducing
		percentChanceWaitTotal += cell.percentChanceWait
		if moment.cells[ci].canopy == true {
			canopyTotal += 1
			grewCanopyAtTotal += moment.cells[ci].grewCanopyAt
		}
		log(LOGTYPE_PRINTGRIDCELLS, "(Cell) %d: %d,%d with %f, age %d, reprod at %f, grew canopy at %f, reproduces with %f\n", ci, moment.cells[ci].x, moment.cells[ci].y, moment.cells[ci].energy, moment.cells[ci].age, moment.cells[ci].energyReproduceThreshold, moment.cells[ci].grewCanopyAt, cell.energySpentOnReproducing)
	}

	log(LOGTYPE_PRINTGRID_SUMMARY, "-----SUMMARY STATE-----\n")
	log(LOGTYPE_PRINTGRID_SUMMARY, "moment %d...\n", momentNum)

	log(LOGTYPE_PRINTGRID_SUMMARY, "%d cells total\n\n", len(moment.cells))
	log(LOGTYPE_MAINLOOPSINGLE, "Cell Count: %d\n", len(currentMoment.cells))
	var energyReproduceThresholdAvg = energyReproduceThresholdTotal / float64(len(moment.cells))
	var grewCanopyAtAvg = grewCanopyAtTotal / float64(canopyTotal)
	var energySpentOnReproducingAvg = energySpentOnReproducingTotal / float64(len(moment.cells))
	var percentChanceWaitAvg = float64(percentChanceWaitTotal) / float64(len(moment.cells))
	log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Reproduce Threshold Average: %f\n", energyReproduceThresholdAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Spent on Reproducing Average: %f\n", energySpentOnReproducingAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Wait Average: %f\n", percentChanceWaitAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Canopy Total: %d\n", canopyTotal)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Grew Canopy At Average: %f\n", grewCanopyAtAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "New species so far: %d\n", speciesCounter)

	log(LOGTYPE_PRINTGRID_SUMMARY, "\n\n\n")
}

const (
	cellaction_reproduce  = iota
	cellaction_growcanopy = iota
	cellaction_wait5      = iota
)

const (
	noncellaction_spontaneouslyPlaceCell = iota
	noncellaction_shineOnAllCells        = iota
	noncellaction_grimReaper             = iota
	noncellaction_cellMaintenance        = iota
)

func generateInitialNonCellActions(wg *sync.WaitGroup) {
	for i := 0; i < initialCellCount; i++ {
		//TODO: is this efficient? Maybe get rid of struct and use raw consts
		queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_spontaneouslyPlaceCell})
		wg.Add(1)
	}
}

func generateSunshineAction(wg *sync.WaitGroup) {
	//TODO for now this action is atom. Could break it up later.
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_shineOnAllCells})
	wg.Add(1)
}

func generateGrimReaperAction(wg *sync.WaitGroup) {
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_grimReaper})
	wg.Add(1)
}

func generateCellMaintenanceAction(wg *sync.WaitGroup) {
	queuedNonCellActions = append(queuedNonCellActions, &NonCellAction{noncellaction_cellMaintenance})
	wg.Add(1)
}

func nonCellActionExecuter(wg *sync.WaitGroup) {
	for {
		var nonCellAction = <-pendingNonCellActions
		//nextMomentLock.Lock()
		//route it to function depending on its type
		switch nonCellAction.actionType {
		case noncellaction_spontaneouslyPlaceCell:
			spontaneouslyGenerateCell()
		case noncellaction_shineOnAllCells:
			shineOnAllCells()
		case noncellaction_grimReaper:
			reapAllDeadCells()
		case noncellaction_cellMaintenance:
			//TODO
			maintainAllCells()
		}

		wg.Done()
		//nextMomentLock.Unlock()
	}
}

//var REPRODUCE_ENERGY_THRESHOLD = 100.0
func (cell *Cell) shouldWaitFive() bool {
	return rand.Intn(100) < cell.percentChanceWait
}

func (cell *Cell) isTimeToReproduce() bool {
	var isThereASpotToReproduce = false
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = cell.x + relativeX
			var yTry = cell.y + relativeY
			if !isOccupied(xTry, yTry, nextMoment) {
				isThereASpotToReproduce = true
				goto foundSpot
			}
		}
	}

foundSpot:
	if !isThereASpotToReproduce {
		return false
	}
	return cell.energy > cell.energyReproduceThreshold
}

const THINKING_COST = 1

func cellActionDecider(wg *sync.WaitGroup) {
	for {

		var cell = <-cellsReadyForAction
		if cell.timeLeftToWait > 0 {
			cell.timeLeftToWait -= 1
		} else {
			if cell.shouldWaitFive() {
				cellActionExecuterWG.Add(1)
				//	fmt.Println("Waitin 5!")
				pendingCellActions <- &CellAction{cell, cellaction_wait5}
			} else if cell.isTimeToReproduce() {
				cellActionExecuterWG.Add(1)
				pendingCellActions <- &CellAction{cell, cellaction_reproduce}
			} else if cell.isReadyToGrowCanopy(cell) {
				cellActionExecuterWG.Add(1)
				pendingCellActions <- &CellAction{cell, cellaction_growcanopy}
			} else {
				//no action at all. Hopefully don't need to submit a null action
			}
			cell.nextMomentSelf.energy -= THINKING_COST
		}
		wg.Done()
		//nextMomentLock.Unlock()
	}
}

func cellActionExecuter(wg *sync.WaitGroup) {
	for {

		var cellAction = <-pendingCellActions
		//nextMomentLock.Lock()
		//TODO: Can these things really get away with never locking?
		switch cellAction.actionType {
		case cellaction_reproduce:
			reproduce(cellAction.cell)
		case cellaction_growcanopy:
			growCanopy(cellAction.cell)
		case cellaction_wait5:
			cellWait5(cellAction.cell)
		}

		wg.Done()
		//nextMomentLock.Unlock()
	}
}

func cellWait5(cell *Cell) {
	cell.nextMomentSelf.timeLeftToWait = 5
}

const REPRODUCE_COST = 30

const (
	LOGTYPE_MAINLOOPSINGLE         = iota
	LOGTYPE_MAINLOOPSINGLE_PRIMARY = iota
	LOGTYPE_FINALSTATS             = iota
	LOGTYPE_HIGHFREQUENCY          = iota
	LOGTYPE_DEBUGCONCURRENCY       = iota
	LOGTYPE_PRINTGRIDCELLS         = iota
	LOGTYPE_PRINTGRID_GRID         = iota
	LOGTYPE_PRINTGRID_SUMMARY      = iota
	LOGTYPE_OTHER                  = iota
	LOGTYPE_FAILURES               = iota
	LOGTYPE_SPECIALEVENT           = iota
)

const printGridEveryNTurns = 50

var logTypesEnabled = []int{LOGTYPE_PRINTGRID_GRID, LOGTYPE_PRINTGRID_SUMMARY, LOGTYPE_FINALSTATS}

func log(logType int, message string, params ...interface{}) {
	if containsInt(logTypesEnabled, logType) {
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

type Direction struct {
	x int
	y int
}

var surroundingDirections = []Direction{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

func getSurroundingDirectionsInRandomOrder() []Direction {
	var surroundingDirectionsInRandomOrder = []Direction{}
	for _, i := range rand.Perm(len(surroundingDirections)) {
		surroundingDirectionsInRandomOrder = append(surroundingDirectionsInRandomOrder, surroundingDirections[i])
	}
	return surroundingDirectionsInRandomOrder
}

func reproduce(cell *Cell) {
	if cell.energy < REPRODUCE_COST || cell.energy < cell.energySpentOnReproducing {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.nextMomentSelf.energy -= cell.energySpentOnReproducing
		return
	}
	//lockAllYs("reproduce")
	//try all spots surrounding the cell
	//lockYRangeInclusive(cell.y-1, cell.y+1, "reproduce")
	for _, direction := range getSurroundingDirectionsInRandomOrder() {
		var xTry = cell.x + direction.x
		var yTry = cell.y + direction.y
		//for relativeX := -1; relativeX < 2; relativeX++ {
		//	for relativeY := -1; relativeY < 2; relativeY++ {
		//		var xTry = cell.x + relativeX
		//		var yTry = cell.y + relativeY

		if !isOccupied(xTry, yTry, nextMoment) {

			log(LOGTYPE_HIGHFREQUENCY, "      cell at %d, %d making a baby\n", cell.x, cell.y)
			var babyCell = cellPool.Borrow()
			babyCell.energy = cell.energySpentOnReproducing - REPRODUCE_COST
			babyCell.energyReproduceThreshold = cell.energyReproduceThreshold + float64(rand.Intn(7)-3)
			babyCell.x = xTry
			babyCell.y = yTry
			babyCell.speciesColor = cell.speciesColor
			babyCell.timeLeftToWait = 0
			//TODO: Better way to do this
			babyCell.percentChanceWait = int(math.Max(0.0, float64(cell.percentChanceWait+rand.Intn(7)-3)))
			babyCell.age = 0
			babyCell.canopy = false
			babyCell._originalEnergyReproduceThreshold = cell._originalEnergyReproduceThreshold
			babyCell._originalEnergySpentOnReproducing = cell._originalEnergySpentOnReproducing
			babyCell._originalGrewCanopyAt = cell._originalGrewCanopyAt
			babyCell._originalPercentChanceWait = cell._originalPercentChanceWait
			babyCell.isReadyToGrowCanopy = makeCanopyBuildConditionOnCertainEnergy(cell.grewCanopyAt + float64(rand.Intn(11)-5))
			babyCell.energySpentOnReproducing = math.Min(babyCell.energyReproduceThreshold, cell.energySpentOnReproducing+float64(rand.Intn(7)-3))

			//TODO: WTF this shit is hella bugged in some mysterious way
			if hasSignificantGeneticDivergence(babyCell) {
				babyCell.speciesColor = getNextColor()
				babyCell._originalEnergyReproduceThreshold = babyCell.energyReproduceThreshold
				babyCell._originalGrewCanopyAt = babyCell.grewCanopyAt
				babyCell._originalEnergySpentOnReproducing = babyCell.energySpentOnReproducing
				log(LOGTYPE_SPECIALEVENT, "Cell at %d, %d "+babyCell.speciesColor.startSequence+"x"+babyCell.speciesColor.endSequence+" is the first of a new species!\n", xTry, yTry)
				log(LOGTYPE_SPECIALEVENT, "orig reprod threshold: %f, new reprod threshold: %f\n", cell._originalEnergyReproduceThreshold, babyCell._originalEnergyReproduceThreshold)
				log(LOGTYPE_SPECIALEVENT, "orig reprod energy spend: %f, new reprod energy spend: %f\n", cell._originalEnergySpentOnReproducing, babyCell._originalEnergySpentOnReproducing)
				log(LOGTYPE_SPECIALEVENT, "orig grow canopy threshold: %f, new grow canopy threshold: %f\n", cell._originalGrewCanopyAt, babyCell._originalGrewCanopyAt)

			}
			nextMoment.cells = append(nextMoment.cells, babyCell)
			nextMoment.cellsSpatialIndex[xTry][yTry] = babyCell
			cell.nextMomentSelf.energy -= cell.energySpentOnReproducing
			return
		}
	}
	//	unlockYRangeInclusive(cell.y-1, cell.y+1, "reproduce")
}

const SPECIES_DIVERGENCE_THRESHOLD = 55

var speciesCounter = 0

func hasSignificantGeneticDivergence(cell *Cell) bool {
	var energyReproduceThresholdDiff = math.Abs(cell._originalEnergyReproduceThreshold - cell.energyReproduceThreshold)
	var grewCanopyAtDiff = 0.0
	if cell.grewCanopyAt != 0.0 {
		grewCanopyAtDiff = math.Abs(cell._originalGrewCanopyAt - cell.grewCanopyAt)
	}
	var energySpentOnReproducingDiff = math.Abs(cell._originalEnergySpentOnReproducing - cell.energySpentOnReproducing)
	var percentChanceWaitDiff = math.Abs(float64(cell._originalPercentChanceWait) - float64(cell.percentChanceWait))
	var totalDiff = energyReproduceThresholdDiff + grewCanopyAtDiff + energySpentOnReproducingDiff + percentChanceWaitDiff
	//if totalDiff > SPECIES_DIVERGENCE_THRESHOLD {
	//	log("original energy threshold: %f\n", cell._originalEnergyReproduceThreshold)
	//	fmt.Printf("current energy threshold: %f\n", cell.energyReproduceThreshold)
	//	fmt.Printf("totalDiff: %f\n", totalDiff)
	//	fmt.Printf("energy spent reprod diff: %f\n", energySpentOnReproducingDiff)
	//	fmt.Printf("grew canopy at diff: %f\n", grewCanopyAtDiff)
	//	fmt.Printf("energy threshold diff: %f\n", energyReproduceThresholdDiff)
	//}
	return totalDiff > SPECIES_DIVERGENCE_THRESHOLD
}

const GROWCANOPY_COST = 50

func growCanopy(cell *Cell) {
	if cell.energy < GROWCANOPY_COST {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.nextMomentSelf.energy -= GROWCANOPY_COST
		return
	} else {
		if cell.grewCanopyAt == 0.0 {
			cell._originalGrewCanopyAt = cell.energy
		}
		cell.nextMomentSelf.grewCanopyAt = cell.energy
		cell.nextMomentSelf.canopy = true
		cell.nextMomentSelf.energy -= GROWCANOPY_COST
	}
}

func maintainAllCells() {
	log(LOGTYPE_MAINLOOPSINGLE, "Starting maintain\n")
	lockAllYs("maintain cells")
	for ci := range nextMoment.cells {
		nextMoment.cells[ci].Maintain()
	}
	unlockAllYs("maintain cells")
	log(LOGTYPE_MAINLOOPSINGLE, "Ending maintain\n")
}

func reapAllDeadCells() {
	log(LOGTYPE_MAINLOOPSINGLE, "Starting reap\n")
	lockAllYs("reaper")
	var cellsToDelete = []*Cell{}
	for ci := range nextMoment.cells {
		var cell = nextMoment.cells[ci]
		if cell.energy <= 0.0 {
			cellsToDelete = append(cellsToDelete, cell)
		}
	}
	if len(cellsToDelete) > 0 {
		nextMoment.RemoveCells(cellsToDelete)
	}
	unlockAllYs("reaper")
	log(LOGTYPE_MAINLOOPSINGLE, "Ending reap\n")
}

func lockAllYs(who string) {
	lockYRangeInclusive(0, GRID_HEIGHT-1, who)
}

func lockYRangeInclusive(start int, end int, who string) {
	log(LOGTYPE_DEBUGCONCURRENCY, "%s Trying to grab bulk lock\n", who)
	bulkGrabLock.Lock()
	log(LOGTYPE_DEBUGCONCURRENCY, "%s Grabbed successfully\n", who)
	for i := start; i < end+1; i++ {
		if !isYOutOfBounds(i, nextMoment) {
			log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to lock %d\n", who, i)
			nextMomentYLocks[i].Lock()
		}
	}
	log(LOGTYPE_DEBUGCONCURRENCY, "%s trying to release bulk lock\n", who)
	bulkGrabLock.Unlock()
}

func unlockYRangeInclusive(start int, end int, who string) {
	for i := start; i < end+1; i++ {
		if !isYOutOfBounds(i, nextMoment) {
			log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to unlock %d\n", who, i)
			nextMomentYLocks[i].Unlock()
		}
	}
}

func unlockAllYs(who string) {
	unlockYRangeInclusive(0, GRID_HEIGHT-1, who)
}

var START_CELL_ENERGY = 50.0
var MAX_TRIES = 100

func spontaneouslyGenerateCell() {
	//TODO: Shouldn't this be next moment?
	//TODO: Probably need to lock some stuff here
	//TODO: Cell pool should probaby zero out cell values before handing it off
	var newCell = cellPool.Borrow()
	newCell.energy = START_CELL_ENERGY
	var foundSpotYet = false
	var tries = 0
	var giveUp = false
	//TODO: Add a timeout to this
	for !foundSpotYet || giveUp {

		var xTry = rand.Intn(GRID_WIDTH)
		var yTry = rand.Intn(GRID_HEIGHT)
		nextMomentYLocks[yTry].Lock()
		if !isOccupied(xTry, yTry, currentMoment) {

			//TODO: This aint gonna work for some reason
			//	nextMomentLock.Lock()
			newCell.x = xTry
			newCell.y = yTry
			newCell.speciesColor = getNextColor()
			newCell.energy = float64(rand.Intn(70))
			newCell.timeLeftToWait = 0

			newCell.percentChanceWait = rand.Intn(40)

			newCell.energySpentOnReproducing = REPRODUCE_COST + float64(rand.Intn(20))
			newCell.energyReproduceThreshold = newCell.energySpentOnReproducing + float64(rand.Intn(120))
			newCell.canopy = false
			newCell.isReadyToGrowCanopy = makeCanopyBuildConditionOnCertainEnergy(float64(rand.Intn(150)) + GROWCANOPY_COST)

			newCell._originalPercentChanceWait = newCell.percentChanceWait
			newCell._originalEnergyReproduceThreshold = newCell.energyReproduceThreshold
			newCell._originalEnergySpentOnReproducing = newCell.energySpentOnReproducing

			nextMoment.cells = append(nextMoment.cells, newCell)
			nextMoment.cellsSpatialIndex[xTry][yTry] = newCell
			foundSpotYet = true
		}
		nextMomentYLocks[yTry].Unlock()
		tries += 1
		if tries > MAX_TRIES {
			log(LOGTYPE_FAILURES, "Gave up on placing tell. Too many cells occupied.")
			break
		}
	}
}

type TextColorBookend struct {
	startSequence string
	endSequence   string
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
				//log(LOGTYPE_FINALSTATS, textColorStart+" BLOOP "+textColorEnd)
				var textColorBookend = &TextColorBookend{textColorStart, textColorEnd}
				textColorBookendsTemp = append(textColorBookendsTemp, textColorBookend)
				//log(LOGTYPE_FINALSTATS, "%d;%d;%d: \033[%d;%d;%dm Hello, World! \033[m \n", i, j, k, i, j, k)
			}
		}
	}
	return textColorBookendsTemp
}

func getNextColor() *TextColorBookend {
	//TODO this is a hack
	speciesCounter++
	var nextColor = textColorBookends[colorCounter]
	colorCounter++
	if colorCounter >= len(textColorBookends) {
		colorCounter = 0
	}
	return nextColor
}

func printAllColors() {
	for ci := range textColorBookends {
		fmt.Println(textColorBookends[ci].startSequence + "Hello, World!" + textColorBookends[ci].endSequence + "\n")
		fmt.Printf(textColorBookends[ci].startSequence + "Hello, World!" + textColorBookends[ci].endSequence + "\n")
	}
}

var SHINE_ENERGY_AMOUNT = 2.5

func shineOnAllCells() {
	log(LOGTYPE_MAINLOOPSINGLE, "Starting shine\n")
	//TODO: This could stand to be refactored a bit
	//TODO: Changed for nextMoment to current moment, but might be wrong...
	for yi := range currentMoment.cellsSpatialIndex {
		lockYRangeInclusive(yi-1, yi+1, "shiner")
		for xi := range currentMoment.cellsSpatialIndex[yi] {
			var cell = currentMoment.cellsSpatialIndex[yi][xi]
			if cell == nil {
				continue
			}
			//fmt.Printf("Proximity to middle: %d\n", int(YProximityToMiddleAsPercent(yi)*100))
			if (momentNum % 100) <= 50 { //int(YProximityToMiddleAsPercent(yi)*100)

				//TODO: should this be next moment?
				var shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * (float64(xi) / float64(GRID_HEIGHT)) //  GRADIENT
				//var shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * float64(float64(yi)/float64(GRID_HEIGHT))
				//fmt.Printf("shine amount at %d: %f", yi, shineAmountForThisSquare)
				// var shineAmountForThisSquare = 0.0
				// if xi%10 == 0 || (xi+1)%10 == 0 || (yi%10 == 0) || ((yi+1)%10) == 0 {
				// 	shineAmountForThisSquare = 0.6
				// } else {
				// 	shineAmountForThisSquare = SHINE_ENERGY_AMOUNT
				// }
				var surroundingCells = getSurroundingCells(cell.x, cell.y, currentMoment)
				var surroundingCellsWithCanopy = []*Cell{}
				for _, surroundingCell := range surroundingCells {
					if surroundingCell.canopy == true {
						surroundingCellsWithCanopy = append(surroundingCellsWithCanopy, surroundingCell)
					}
				}
				var surroundingCellsWithCanopyAndMe = append(surroundingCellsWithCanopy, cell)
				var energyToEachCell = shineAmountForThisSquare / float64(len(surroundingCellsWithCanopyAndMe))
				for _, cellToReceiveEnergy := range surroundingCellsWithCanopyAndMe {
					//TODO: Is there something fishy here?
					if cellToReceiveEnergy.nextMomentSelf != nil {
						cellToReceiveEnergy.nextMomentSelf.energy += energyToEachCell
					}
				}
			} else {
				//No sun at night
			}
			log(LOGTYPE_HIGHFREQUENCY, "Shiner touching on %d, %d \n", xi, yi)
		}
		unlockYRangeInclusive(yi-1, yi+1, "shiner")
	}
	log(LOGTYPE_MAINLOOPSINGLE, "Ending shine\n")
}

func YProximityToMiddleAsPercent(y int) float64 {
	//fmt.Printf("supposed y: %d", y)
	var middle = GRID_WIDTH / 2.0
	//fmt.Printf("middle: %f", middle)
	var proximityToMiddle = math.Abs(float64(y) - middle)
	//	fmt.Printf("proximityToMiddle: %f\n", proximityToMiddle)
	var proximityToMiddleAsPercent = ((middle - proximityToMiddle) / middle)
	//fmt.Printf("proximityToMiddleAsPercent: %f\n", proximityToMiddleAsPercent)
	return proximityToMiddleAsPercent
}

func getSurroundingCells(x int, y int, moment *Moment) []*Cell {
	var surroundingCells []*Cell
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = x + relativeX
			var yTry = y + relativeY
			if !isOutOfBounds(xTry, yTry, moment) && isOccupied(xTry, yTry, moment) {
				var cell = moment.cellsSpatialIndex[xTry][yTry]
				surroundingCells = append(surroundingCells, cell)
			}
		}
	}
	return surroundingCells
}

func isOccupied(x int, y int, moment *Moment) bool {
	if isOutOfBounds(x, y, moment) {
		return true
	} else {
		return moment.cellsSpatialIndex[x][y] != nil
	}
}

func isOutOfBounds(x int, y int, moment *Moment) bool {
	return isXOutOfBounds(x, moment) || isYOutOfBounds(y, moment)
}

func isXOutOfBounds(x int, moment *Moment) bool {
	return x < 0 || x > GRID_WIDTH-1
}

func isYOutOfBounds(y int, moment *Moment) bool {
	return y < 0 || y > GRID_HEIGHT-1
}
