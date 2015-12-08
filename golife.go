package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/davecheney/profile"
)

//FOUNDATIONAL WORLD VARIABLES

//---RUN_CONDITIONS-----
const RANDOM_SEED = true

const MAX_MOMENTS = 9000

//---WORLD_CONDITIONS---
const SHINE_ENERGY_AMOUNT = 2.0

const INITIAL_CELL_COUNT = 200

const GRID_WIDTH = 93
const GRID_HEIGHT = 450

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

//UI VARIABLES
const LOGTYPES_ENABLED = []int{LOGTYPE_PRINTGRID_BIGGRID, LOGTYPE_PRINTGRID_SUMMARY, LOGTYPE_SPECIESREPORT, LOGTYPE_PRINTGRID_SUMMARY, LOGTYPE_FINALSTATS}

const PRINTGRID_EVERY_N_TURNS = 10

const SPECIES_DIVERGENCE_THRESHOLD = 65

const BIGGRID_INCREMENT = 3

const NUM_TOP_SPECIES_TO_PRINT = 10

const ACTUAL_WAIT_MULTIPLIER = 3

const MAX_TRIES_TO_FIND_EMPTY_GRID_COORD = 100

//---PERFORMANCE_VARIABLES---
const CELLACTIONDECIDER_ROUTINECOUNT = 25
const CELLACTIONDECIDER_ROUTINECOUNT = 1
const NONCELLACTIONDECIDER_ROUTINECOUNT = 1

const MAX_CELL_COUNT = 900000

const CELLSATATIME_PER_DECIDER = 1000

//END CONSTANTS-x-x-x-x-x-x-x-x-x-x-x

var secretIDCounter = 0

func newCell() *Cell {
	secretIDCounter++
	return &Cell{_secretID: secretIDCounter}
}

type Pool struct {
	pool chan *Cell
}

const SURROUNDINGS_SIZE = 9

type SurroundingsPool struct {
	pool chan *[SURROUNDINGS_SIZE]*Cell
}

const CELLS_PER_BUNDLE = 500

type CellBundle struct {
	cells []*Cell
}

type CellActionBundle struct {
	actions []*CellAction
}

// NewPool creates a new pool of Cells
func newPool(max int) *Pool {
	return &Pool{
		pool: make(chan *Cell, max),
	}
}

func newSurroundingsPool(max int) *SurroundingsPool {
	return &SurroundingsPool{
		pool: make(chan *[SURROUNDINGS_SIZE]*Cell, max),
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

func (p *SurroundingsPool) Borrow() *[SURROUNDINGS_SIZE]*Cell {
	var c *[SURROUNDINGS_SIZE]*Cell
	select {
	case c = <-p.pool:
	default:
		c = newSurroundings()
	}
	return c
}

// Return returns a Cell to the pool.
//TODO: Should all these be pointers to slices rather than slices? *[]*Cell instead
func (p *SurroundingsPool) Return(c *[SURROUNDINGS_SIZE]*Cell) {
	select {
	case p.pool <- c:
	default:

	}
}

func (p *Pool) Return(c *Cell) {
	select {
	case p.pool <- c:
	default:

	}
}

func newSurroundings() *[SURROUNDINGS_SIZE]*Cell {
	return &[SURROUNDINGS_SIZE]*Cell{}
}

//TODO: Make it easy to add a field and have it appear in all the right places Re: copying and whatnot
type Cell struct {
	_energy                           float64
	dead                              bool
	x                                 int
	y                                 int
	id                                int
	_timeLeftToWait                   int
	clockRate                         int
	canopy                            bool
	legs                              bool
	growCanopyAt                      float64
	growLegsAt                        float64
	energySpentOnReproducing          float64
	nextMomentSelf                    *Cell
	age                               int
	speciesID                         int
	percentChanceWait                 int     //out of 100
	moveChance                        float64 //out of 100
	_secretID                         int
	_originalgrowCanopyAt             float64
	_originalPercentChanceWait        int
	_originalEnergySpentOnReproducing float64
	_originalEnergyReproduceThreshold float64
	_originalClockRate                int
	_originalGrowLegsAt               float64
	_originalMoveChance               float64
	energyReproduceThreshold          float64
	speciesColor                      *TextColorBookend
}

func (cell *Cell) decreaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	if (!(cell._energy <= 0.0)) || cell.nextMomentSelf != nil {
		//	if !cell.isDead() {
		//log(LOGTYPE_CELLEFFECT, "the future cell %d currently has %f energy\n", cell.nextMomentSelf.id, cell.nextMomentSelf._energy)
		//log(LOGTYPE_CELLEFFECT, "decreased cell %d by %f\n", cell.nextMomentSelf.id, amt)
		cell.nextMomentSelf._energy = cell.nextMomentSelf._energy - amt
		//log(LOGTYPE_CELLEFFECT, "cell %d will now have %f energy\n", cell.nextMomentSelf.id, cell.nextMomentSelf._energy)
	}
}

func (cell *Cell) increaseEnergy(amt float64) {
	//TODO: Inlined dead stuff for performance reasons
	if (!(cell._energy <= 0.0)) || cell.nextMomentSelf != nil {
		//	log(LOGTYPE_CELLEFFECT, "the future cell %d currently has %f energy\n", cell.nextMomentSelf.id, cell.nextMomentSelf._energy)
		//	log(LOGTYPE_CELLEFFECT, "increased cell %d by %f\n", cell.nextMomentSelf.id, amt)
		cell.nextMomentSelf._energy = cell.nextMomentSelf._energy + amt

		//	log(LOGTYPE_CELLEFFECT, "cell %d will now have %f energy\n", cell.nextMomentSelf.id, cell.nextMomentSelf._energy)
	}
}

func (oldCell *Cell) Copy() *Cell {
	//TODO: Can this not be done by just a generic struct copy?
	var newCell = cellPool.Borrow()
	newCell._energy = oldCell._energy
	newCell.age = oldCell.age
	newCell.x = oldCell.x
	newCell.id = oldCell.id
	newCell.y = oldCell.y
	newCell.speciesID = oldCell.speciesID
	newCell.percentChanceWait = oldCell.percentChanceWait
	newCell.speciesColor = oldCell.speciesColor
	newCell.energySpentOnReproducing = oldCell.energySpentOnReproducing
	newCell.canopy = oldCell.canopy
	newCell.clockRate = oldCell.clockRate
	newCell.growCanopyAt = oldCell.growCanopyAt
	newCell.energyReproduceThreshold = oldCell.energyReproduceThreshold
	newCell.legs = oldCell.legs
	newCell.moveChance = oldCell.moveChance
	newCell.growLegsAt = oldCell.growLegsAt
	newCell._originalMoveChance = oldCell._originalMoveChance
	newCell._originalGrowLegsAt = oldCell._originalGrowLegsAt
	newCell._originalEnergyReproduceThreshold = oldCell._originalEnergyReproduceThreshold
	newCell._originalEnergySpentOnReproducing = oldCell._originalEnergySpentOnReproducing
	newCell._originalgrowCanopyAt = oldCell._originalgrowCanopyAt
	newCell._originalPercentChanceWait = oldCell._originalPercentChanceWait
	newCell._originalClockRate = oldCell._originalClockRate

	return newCell
}

func (oldCell *Cell) ContinueOn() *Cell {
	var continuedCell = oldCell.Copy()
	log(LOGTYPE_CELLEFFECT, "cell %d now has a future self established during moment %d\n", oldCell.id, momentNum)
	oldCell.nextMomentSelf = continuedCell
	return continuedCell
}

func (cell *Cell) Maintain() {
	var totalUpkeep = BASE_CELL_UPKEEP
	if cell.legs {
		totalUpkeep += LEGS_UPKEEP
	}
	if cell.canopy {
		totalUpkeep += CANOPY_UPKEEP
	}
	log(LOGTYPE_CELLEFFECT, "cell %d is about to be maintained\n", cell.id)
	cell.decreaseEnergy((totalUpkeep * float64(cell.age)) / CELL_LIFESPAN)
	//cell.decreaseEnergy(40)
	//	cell._energy -= totalUpkeep * float64(cell.age) / CELL_LIFESPAN
	cell.increaseAge(1)
}

func (cell *Cell) increaseAge(amt int) {
	if !cell.isDead() {
		cell.nextMomentSelf.age = cell.nextMomentSelf.age + amt
	}
}

type CellAction struct {
	cell       *Cell
	actionType int
}

type NonCellAction struct {
	actionType int
}

//const GRID_WIDTH = 1
//const GRID_HEIGHT = 100

type Moment struct {
	cells             []*Cell
	cellsSpatialIndex [GRID_WIDTH][GRID_HEIGHT]*Cell
}

func (moment *Moment) RemoveCells(cellsToDelete []*Cell) {
	//TODO: Wow I bet this performance sucks. Can we do better?

	//w := 0 // write index
	data := make([]*Cell, 0, len(moment.cells)-len(cellsToDelete))
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
var surroundingsPool *SurroundingsPool

var oldMoment *Moment
var momentBeingCleaned *Moment
var momentReadyToBeNext *Moment
var currentMoment *Moment
var nextMoment *Moment

//var nextMomentLock *sync.Mutex

//var nextMomentYXLocks [GRID_WIDTH][GRID_HEIGHT]sync.Mutex
var bulkGrabLock *sync.Mutex

var cellActionExecuterWG sync.WaitGroup

//var cellsReadyForAction = make(chan *CellBundle, MAX_CELL_COUNT)
//var cellsReadyForAction = make(chan *Cell, MAX_CELL_COUNT)
var cellsReadyForAction = make(chan []*Cell, MAX_CELL_COUNT)

var queuedNonCellActions []*NonCellAction
var pendingNonCellActions = make(chan *NonCellAction, MAX_CELL_COUNT)

//var pendingCellActions = make(chan *CellActionBundle, MAX_CELL_COUNT)
var pendingCellActions = make(chan *CellAction, MAX_CELL_COUNT)

var cellActionDeciderWG sync.WaitGroup
var nonCellActionExecuterWG sync.WaitGroup

var waitForCleaning sync.WaitGroup

func createThisManyCells(startHere int, endBeforeHere int, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := startHere; i < endBeforeHere; i++ {
		cellPool.Borrow()
	}
}

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
	for i := 0; i < CELLACTIONDECIDER_ROUTINECOUNT; i++ {
		go cellActionExecuter(&cellActionExecuterWG)
	}
}

func transferLiveCellsToNextMoment() {
	nextMoment.cells = make([]*Cell, 0, len(currentMoment.cells))
	nextMoment.cellsSpatialIndex = [GRID_WIDTH][GRID_HEIGHT]*Cell{}
	for _, currentMomentCell := range currentMoment.cells {
		//This takes place of reaper function
		//log(LOGTYPE_HIGHFREQUENCY, "a cell %d, has %6.2f energy\n", currentMomentCell._secretID, currentMomentCell._energy)
		if currentMomentCell._energy > 0 {
			var nextMomentCell = currentMomentCell.ContinueOn()
			nextMoment.cells = append(nextMoment.cells, nextMomentCell)
			nextMoment.cellsSpatialIndex[nextMomentCell.x][nextMomentCell.y] = nextMomentCell
		}
	}
	log(LOGTYPE_MAINLOOPSINGLE, "Transferred cells over to next moment by default, same loc\n")
}

func feedCurrentMomentCellsToActionDecider() {
	var startSlice = 0
	var endSlice = CELLS_PER_BUNDLE
	for {
		if endSlice < len(currentMoment.cells) {
			cellActionDeciderWG.Add(1)
			cellsReadyForAction <- currentMoment.cells[startSlice:endSlice]
			startSlice += CELLS_PER_BUNDLE
			endSlice += CELLS_PER_BUNDLE
		} else {
			cellActionDeciderWG.Add(1)
			cellsReadyForAction <- currentMoment.cells[startSlice:]
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

	cellPool = newPool(MAX_CELL_COUNT)
	surroundingsPool = newSurroundingsPool(MAX_CELL_COUNT)
	currentMoment = &Moment{}
	nextMoment = &Moment{}
	momentBeingCleaned = &Moment{}

	//nextMomentYXLocks = [GRID_WIDTH][GRID_HEIGHT]sync.Mutex{}
	bulkGrabLock = &sync.Mutex{}

	startPersistentThreads()

	var t1all = time.Now()
	for momentNum = 0; momentNum < MAX_MOMENTS; momentNum++ {
		if momentNum%PRINTGRID_EVERY_N_TURNS == 0 {
			printGrid(currentMoment)
			printSpeciesReport(currentMoment, NUM_TOP_SPECIES_TO_PRINT)
		}
		var t1a = time.Now()
		//var t1 = time.Now()
		log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "moment %d...\n", momentNum)
		//Assume all cells will be in same position in next moment
		//TODO: Should this be happening elsewhere?
		transferLiveCellsToNextMoment()

		feedCurrentMomentCellsToActionDecider()

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
			log(LOGTYPE_MAINLOOPSINGLE, "Generating cell maintenance and other stuff\n")
			generateSunshineAction(&nonCellActionExecuterWG)
			generateCellMaintenanceAction(&nonCellActionExecuterWG)
			//TODO: Testing reap removal
			//generateGrimReaperAction(&nonCellActionExecuterWG)
		}
		//feed waitingNonCellActions to readyNonCellActions
		log(LOGTYPE_MAINLOOPSINGLE, "Dumping non-cell actions into working queue (if any)\n")

		feedQueuedNonCellActionsToExecuter()

		log(LOGTYPE_MAINLOOPSINGLE, "Transferred to non-cell executers...\n")
		//wait for all non-cell actions to be done
		nonCellActionExecuterWG.Wait()

		log(LOGTYPE_MAINLOOPSINGLE, "Non-cell Executers did their thing\n")
		//nextMoment becomes current moment.
		log(LOGTYPE_MAINLOOPSINGLE, "Right before we switch to next moment")
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
		log(LOGTYPE_MAINLOOPSINGLE, "Right after the switcheroo")
		log(LOGTYPE_MAINLOOPSINGLE, "About to wait for cleaning\n")
		waitForCleaning.Add(1)
		momentBeingCleaned.ReturnCellsToPool()
		go momentBeingCleaned.Clean(&waitForCleaning)

		var dura = t2a.Sub(t1a).Seconds()
		log(LOGTYPE_MAINLOOPSINGLE_PRIMARY, "Time of entire turn took %f\n", dura)
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
		return cell.canopy == false && cell._energy > makeCanopyEnergyThreshold
	}
	return buildConditionFunction
}

func printCell(cell *Cell) {
	if cell != nil {
		var colorStart = cell.speciesColor.startSequence
		var colorEnd = cell.speciesColor.endSequence
		if cell.canopy == true {
			log(LOGTYPE_PRINTGRID_GRID, colorStart+"X"+colorEnd)
			log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"X"+colorEnd)
		} else {
			log(LOGTYPE_PRINTGRID_GRID, colorStart+"x"+colorEnd)
			log(LOGTYPE_PRINTGRID_BIGGRID, colorStart+"x"+colorEnd)
		}
	} else {
		log(LOGTYPE_PRINTGRID_GRID, " ")
	}
}

func printGrid(moment *Moment) {
	if containsInt(LOGTYPES_ENABLED, LOGTYPE_PRINTGRID_GRID) {
		log(LOGTYPE_PRINTGRID_GRID, "\n")
		for row := range moment.cellsSpatialIndex {
			for col := range moment.cellsSpatialIndex[row] {
				var cell = moment.cellsSpatialIndex[row][col]
				printCell(cell)
			}
			log(LOGTYPE_PRINTGRID_GRID, "\n")
		}
	} else if containsInt(LOGTYPES_ENABLED, LOGTYPE_PRINTGRID_BIGGRID) {
		log(LOGTYPE_PRINTGRID_BIGGRID, "\n")
		for row := 0; row < len(moment.cellsSpatialIndex); row += BIGGRID_INCREMENT {
			for col := 0; col < len(moment.cellsSpatialIndex); col += BIGGRID_INCREMENT {
				var cell = moment.cellsSpatialIndex[row][col]
				printCell(cell)
			}
			log(LOGTYPE_PRINTGRID_BIGGRID, "\n")
		}
	}
	log(LOGTYPE_PRINTGRID_GRID, "\n")

	var energyReproduceThresholdTotal = 0.0
	var energySpentOnReproducingTotal = 0.0
	var canopyTotal = 0
	var growCanopyAtTotal = 0.0
	var percentChanceWaitTotal = 0
	var clockRateTotal = 0
	var moveChanceTotal = 0.0
	var legsTotal = 0
	for ci := range moment.cells {
		var cell = moment.cells[ci]
		//TODO: Refactor all this crap to use cell variable
		energyReproduceThresholdTotal += cell.energyReproduceThreshold
		energySpentOnReproducingTotal += cell.energySpentOnReproducing
		percentChanceWaitTotal += cell.percentChanceWait
		clockRateTotal += cell.clockRate
		if moment.cells[ci].canopy == true {
			canopyTotal += 1
			growCanopyAtTotal += moment.cells[ci].growCanopyAt
		}
		if cell.legs == true {
			legsTotal += 1
			moveChanceTotal += cell.moveChance
		}
		log(LOGTYPE_PRINTGRIDCELLS, "(Cell) %d: %d,%d with %f, age %d, reprod at %f, grew canopy at %f, reproduces with %f\n", cell.id, moment.cells[ci].x, moment.cells[ci].y, moment.cells[ci]._energy, moment.cells[ci].age, moment.cells[ci].energyReproduceThreshold, moment.cells[ci].growCanopyAt, cell.energySpentOnReproducing)
	}

	log(LOGTYPE_PRINTGRID_SUMMARY, "-----SUMMARY STATE-----\n")
	log(LOGTYPE_PRINTGRID_SUMMARY, "moment %d...\n", momentNum)

	log(LOGTYPE_PRINTGRID_SUMMARY, "%d cells total\n\n", len(moment.cells))
	log(LOGTYPE_MAINLOOPSINGLE, "Cell Count: %d\n", len(currentMoment.cells))
	var energyReproduceThresholdAvg = energyReproduceThresholdTotal / float64(len(moment.cells))
	var growCanopyAtAvg = growCanopyAtTotal / float64(canopyTotal)
	var energySpentOnReproducingAvg = energySpentOnReproducingTotal / float64(len(moment.cells))
	var percentChanceWaitAvg = float64(percentChanceWaitTotal) / float64(len(moment.cells))
	var clockRateAvg = float64(clockRateTotal) / float64(len(moment.cells))
	var percentMoveAvg = float64(moveChanceTotal) / float64(legsTotal)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Reproduce Threshold Average: %f\n", energyReproduceThresholdAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Energy Spent on Reproducing Average: %f\n", energySpentOnReproducingAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Wait Average: %f\n", percentChanceWaitAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Average Clock Rate: %f\n", clockRateAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Canopy Total: %d\n", canopyTotal)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Grew Canopy At Average: %f\n", growCanopyAtAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Legs Total: %d\n", legsTotal)
	log(LOGTYPE_PRINTGRID_SUMMARY, "Percent Chance to Move (with legs) Average: %f\n", percentMoveAvg)
	log(LOGTYPE_PRINTGRID_SUMMARY, "New species so far: %d\n", speciesCounter)

	log(LOGTYPE_PRINTGRID_SUMMARY, "\n\n\n")
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

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func printSpeciesReport(moment *Moment, topXSpecies int) {
	var speciesIDToCount = make(map[string]int)
	var speciesIDToSpecimen = make(map[string]*Cell)
	for ci := range moment.cells {
		var cell = moment.cells[ci]
		var speciesIDString = string(cell.speciesID)
		var speciesCount, exists = speciesIDToCount[speciesIDString]
		if exists {
			speciesIDToCount[speciesIDString] = speciesCount + 1
		} else {
			speciesIDToCount[speciesIDString] = 1
			speciesIDToSpecimen[speciesIDString] = cell
		}
	}

	var speciesIDByCount = rankByCount(speciesIDToCount)
	var realTopXSpecies = int(math.Min(float64(topXSpecies), float64(len(speciesIDByCount))))
	var topXSpeciesIDByCount = speciesIDByCount[:realTopXSpecies]
	log(LOGTYPE_SPECIESREPORT, "\n")
	log(LOGTYPE_SPECIESREPORT, "-----SPECIES REPORT----\n")
	for _, pair := range topXSpeciesIDByCount {
		var speciesID = pair.Key
		var count = speciesIDToCount[speciesID]
		//TODO: This bugged
		var specimen *Cell = speciesIDToSpecimen[speciesID]

		log(LOGTYPE_SPECIESREPORT, "Species %s\t "+specimen.speciesColor.startSequence+"x"+specimen.speciesColor.endSequence+"\t"+"\t"+"Count: %d\t reprod threshold: %6.1f\t reprod energy: %6.1f\t gcanopy thresh: %6.1f\t wait percent: %d\t clock rate %d\t gleg thresh: %6.1f\n",
			speciesID, count, specimen._originalEnergyReproduceThreshold, specimen._originalEnergySpentOnReproducing, specimen._originalgrowCanopyAt, specimen._originalPercentChanceWait, specimen._originalClockRate, specimen._originalGrowLegsAt)
	}
	log(LOGTYPE_SPECIESREPORT, "\n")
}

const (
	cellaction_reproduce  = iota
	cellaction_growcanopy = iota
	cellaction_wait       = iota
	cellaction_growlegs   = iota
	cellaction_moverandom = iota
)

const (
	noncellaction_spontaneouslyPlaceCell = iota
	noncellaction_shineOnAllCells        = iota
	noncellaction_grimReaper             = iota
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
			//TODO: Removing reapar call as a test, doing it implicitly now when next turn starts
			//reapAllDeadCells()
		case noncellaction_cellMaintenance:
			//TODO
			maintainAllCells()
		}

		wg.Done()
		//nextMomentLock.Unlock()
	}
}

//var REPRODUCE_ENERGY_THRESHOLD = 100.0
func (cell *Cell) shouldWait() bool {
	return rand.Intn(100) < cell.percentChanceWait
}

func (cell *Cell) isDead() bool {
	//TODO: Why do I have to check future self as well?

	return cell._energy <= 0 || cell.nextMomentSelf == nil
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
	return cell._energy > cell.energyReproduceThreshold
}

func (cell *Cell) isReadyToGrowCanopy() bool {
	return cell.canopy == false && cell._energy > cell.growCanopyAt
}

func (cell *Cell) wantsToAndCanMove() bool {
	var isThereASpotToMove = false
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = cell.x + relativeX
			var yTry = cell.y + relativeY
			if !isOccupied(xTry, yTry, nextMoment) {
				isThereASpotToMove = true
				goto foundSpot
			}
		}
	}

foundSpot:
	if !isThereASpotToMove {
		return false
	}
	return cell._energy > MOVE_COST
}

func (cell *Cell) isReadyToGrowLegs() bool {
	return cell.legs == false && cell._energy > cell.growLegsAt
}

func (cell *Cell) countDown_timeLeftToWait() {
	if !cell.isDead() {
		cell.nextMomentSelf._timeLeftToWait -= 1
	}
}

var GUESS_AT_PERCENT_CELLS_ACTING = 0.1

func cellActionDecider(wg *sync.WaitGroup) {
	for {

		//	var cellBundle = <-cellsReadyForAction
		//for _, cell := range cellBundle.cells {
		var cellSlice = <-cellsReadyForAction
		//var cellActionBundle = CellActionBundle{}
		//	cellActionBundle.actions = make([]*CellAction, 0, int(float64(len(cellBundle.cells))*GUESS_AT_PERCENT_CELLS_ACTING*2))
		for _, cell := range cellSlice {
			if cell._timeLeftToWait > 0 {
				cell.countDown_timeLeftToWait()
			} else {
				if cell.isReadyToGrowCanopy() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_growcanopy}
					//	cellActionBundle.actions = append(cellActionBundle.actions, &CellAction{cell, cellaction_growcanopy})

				} else if cell.isReadyToGrowLegs() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_growlegs}
					//cellActionBundle.actions = append(cellActionBundle.actions, &CellAction{cell, cellaction_growlegs})

				} else if cell.isTimeToReproduce() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_reproduce}
					//cellActionBundle.actions = append(cellActionBundle.actions, &CellAction{cell, cellaction_reproduce})

				} else if cell.wantsToAndCanMove() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_moverandom}
					//cellActionBundle.actions = append(cellActionBundle.actions, &CellAction{cell, cellaction_moverandom})

				} else if cell.shouldWait() {
					cellActionExecuterWG.Add(1)
					pendingCellActions <- &CellAction{cell, cellaction_wait}
					//cellActionBundle.actions = append(cellActionBundle.actions, &CellAction{cell, cellaction_wait})

				} else {
					//no action at all. Hopefully don't need to submit a null action
				}
				cell.decreaseEnergy(THINKING_COST)
				cell.increaseWaitTime(cell.clockRate - 1)
				//TODO: Make sure not to do off-by-one here. Clock 1 should be every 1 turn
			}
		}
		//cellActionExecuterWG.Add(1)
		//pendingCellActions <- &cellActionBundle
		//}
		wg.Done()
		//nextMomentLock.Unlock()
	}
}

func (cell *Cell) increaseWaitTime(amt int) {
	if !cell.isDead() {
		cell.nextMomentSelf._timeLeftToWait += amt
	}
}

func cellActionExecuter(wg *sync.WaitGroup) {
	for {
		//var cellActionBundle = <-pendingCellActions
		//for _, cellAction := range cellActionBundle.actions
		var cellAction = <-pendingCellActions
		//nextMomentLock.Lock()
		//TODO: Can these things really get away with never locking?
		switch cellAction.actionType {
		case cellaction_reproduce:
			reproduce(cellAction.cell)
		case cellaction_growcanopy:
			cellAction.cell.growCanopy()
		case cellaction_wait:
			cellAction.cell.wait()
		case cellaction_growlegs:
			cellAction.cell.growLegs()
		}
		wg.Done()
	}

	//nextMomentLock.Unlock()
}

func (cell *Cell) growLegs() {

	if cell.isDead() {
		return
	} else if !cell.isReadyToGrowLegs() {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.nextMomentSelf.decreaseEnergy(GROWLEGS_COST)
		return
	} else {
		var nextMomentSelf = cell.nextMomentSelf
		//TODO: Disabled leg growing. Should re-enable when it's no longer under suspicion
		nextMomentSelf.legs = false
		nextMomentSelf.decreaseEnergy(GROWLEGS_COST)
	}
}

func moveRandom(cell *Cell) bool {
	if !cell.wantsToAndCanMove() {
		return false
	}

	for _, direction := range getSurroundingDirectionsInRandomOrder() {
		var xTry = cell.x + direction.x
		var yTry = cell.y + direction.y

		//TODO: Really need to do some locking here to prevent corruption
		if !isOccupied(xTry, yTry, nextMoment) {
			nextMoment.cellsSpatialIndex[xTry][yTry] = cell
			nextMoment.cellsSpatialIndex[cell.x][cell.y] = nil
			cell.x = xTry
			cell.y = yTry
			cell.decreaseEnergy(MOVE_COST)

			return true
		}
	}
	return false
}

func (cell *Cell) wait() {
	if !cell.isDead() {
		cell.nextMomentSelf._timeLeftToWait = ACTUAL_WAIT_MULTIPLIER * cell.clockRate
	}
}

const (
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

func log(logType int, message string, params ...interface{}) {
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
	if cell._energy < REPRODUCE_COST || cell._energy < cell.energySpentOnReproducing {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.decreaseEnergy(cell.energySpentOnReproducing)
		return
	}
	lockAllYXs("reproduce")
	//try all spots surrounding the cell
	//TODO: Why are we not locking here???
	//lockYRangeInclusive(cell.y-1, cell.y+1, "reproduce")
	for _, direction := range getSurroundingDirectionsInRandomOrder() {
		var xTry = cell.x + direction.x
		var yTry = cell.y + direction.y
		//for relativeX := -1; relativeX < 2; relativeX++ {
		//	for relativeY := -1; relativeY < 2; relativeY++ {
		//		var xTry = cell.x + relativeX
		//		var yTry = cell.y + relativeY

		if !isOccupied(xTry, yTry, nextMoment) {

			var rand0To99 = rand.Intn(100)
			log(LOGTYPE_CELLEFFECT, "cell %d at %d, %d making a baby\n", cell.id, cell.x, cell.y)
			var babyCell = cellPool.Borrow()
			babyCell._energy = cell.energySpentOnReproducing - REPRODUCE_COST
			//TODO: This should be a function, probably needs locking if parallelized
			babyCell.id = IDCounter
			IDCounter += 1
			log(LOGTYPE_CELLEFFECT, "baby cell %d born with %f energy\n", babyCell.id, babyCell._energy)
			babyCell.energyReproduceThreshold = cell.energyReproduceThreshold + float64(rand.Intn(7)-3)
			babyCell.x = xTry
			babyCell.y = yTry
			babyCell.speciesID = cell.speciesID
			babyCell.speciesColor = cell.speciesColor

			babyCell._timeLeftToWait = 0
			if rand0To99 < 40 {
				babyCell.clockRate = cell.clockRate + rand.Intn(9) - 4
			} else if rand0To99 < 80 {
				babyCell.clockRate = cell.clockRate + rand.Intn(3) - 1
			} else {
				babyCell.clockRate = cell.clockRate
			}
			//TODO: Better way to do this

			//TODO: This whole way of doing legs is weird. Should be on or off switch methinks
			//var rand0To99_2 = rand.Intn(100)
			//if rand0To99_2 == 0 {
			//legless gets legs
			//	babyCell.growLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST
			//	babyCell.moveChance = float64(rand.Intn(40))
			//} else if rand0To99_2 == 1 {
			//legged goes legless
			//	babyCell.growLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST + 1000
			//	babyCell.moveChance = 0.0
			//} else {
			//mutate like normal
			//	babyCell.growLegsAt = math.Max(GROWLEGS_COST, float64(cell.growLegsAt+float64(rand.Intn(7)-3)))
			//	babyCell.moveChance = math.Max(0.0, float64(cell.moveChance+float64(rand.Intn(7)-3)))
			//}
			//TODO: Just trying to disable this
			babyCell.growLegsAt = 999

			babyCell.percentChanceWait = int(math.Max(0.0, float64(cell.percentChanceWait+rand.Intn(7)-3)))

			babyCell.age = 0
			babyCell.canopy = false
			babyCell.growCanopyAt = cell.growCanopyAt + float64(rand.Intn(7)-3)

			babyCell._originalEnergyReproduceThreshold = cell._originalEnergyReproduceThreshold
			babyCell._originalEnergySpentOnReproducing = cell._originalEnergySpentOnReproducing
			babyCell._originalgrowCanopyAt = cell._originalgrowCanopyAt
			babyCell._originalPercentChanceWait = cell._originalPercentChanceWait
			babyCell._originalClockRate = cell._originalClockRate
			babyCell._originalGrowLegsAt = cell._originalGrowLegsAt
			babyCell._originalMoveChance = cell._originalMoveChance

			babyCell.energySpentOnReproducing = math.Min(babyCell.energyReproduceThreshold, cell.energySpentOnReproducing+float64(rand.Intn(7)-3))

			//TODO: WTF this shit is hella bugged in some mysterious way
			if hasSignificantGeneticDivergence(babyCell) {
				//TODO: This should probably be a function
				babyCell.speciesID = speciesIDCounter
				speciesIDCounter++
				babyCell.speciesColor = getNextColor()

				babyCell._originalEnergyReproduceThreshold = babyCell.energyReproduceThreshold
				babyCell._originalgrowCanopyAt = babyCell.growCanopyAt
				babyCell._originalEnergySpentOnReproducing = babyCell.energySpentOnReproducing
				babyCell._originalPercentChanceWait = babyCell.percentChanceWait
				babyCell._originalClockRate = babyCell.clockRate
				babyCell._originalGrowLegsAt = babyCell.growLegsAt
				babyCell._originalMoveChance = babyCell.moveChance

				log(LOGTYPE_SPECIALEVENT, "Cell at %d, %d "+babyCell.speciesColor.startSequence+"x"+babyCell.speciesColor.endSequence+" is the first of a new species!\n", xTry, yTry)
				log(LOGTYPE_SPECIALEVENT, "orig reprod threshold: %f, new reprod threshold: %f\n", cell._originalEnergyReproduceThreshold, babyCell._originalEnergyReproduceThreshold)
				log(LOGTYPE_SPECIALEVENT, "orig reprod energy spend: %f, new reprod energy spend: %f\n", cell._originalEnergySpentOnReproducing, babyCell._originalEnergySpentOnReproducing)
				log(LOGTYPE_SPECIALEVENT, "orig grow canopy threshold: %f, new grow canopy threshold: %f\n", cell._originalgrowCanopyAt, babyCell._originalgrowCanopyAt)

			}
			nextMoment.cells = append(nextMoment.cells, babyCell)
			nextMoment.cellsSpatialIndex[xTry][yTry] = babyCell
			cell.decreaseEnergy(cell.energySpentOnReproducing)
			return
		}
	}
	//	unlockYRangeInclusive(cell.y-1, cell.y+1, "reproduce")
}

var speciesCounter = 0

func hasSignificantGeneticDivergence(cell *Cell) bool {
	var energyReproduceThresholdDiff = math.Abs(cell._originalEnergyReproduceThreshold - cell.energyReproduceThreshold)
	//TODO: took canopy out because something is jacked up about it. Need to debug and put back in
	//var growCanopyAtDiff = 0.0
	//if cell.growCanopyAt != 0.0 {
	//	growCanopyAtDiff = math.Abs(cell._originalgrowCanopyAt - cell.growCanopyAt)
	//}

	//var moveChanceDiff = math.Abs(float64(cell._originalMoveChance) - float64(cell.moveChance))
	//var growLegsAtDiff = math.Abs(float64(cell._originalGrowLegsAt) - float64(cell.growCanopyAt))
	var growCanopyAtDiff = math.Abs(float64(cell._originalgrowCanopyAt) - float64(cell.growCanopyAt))
	var clockRateDiff = math.Abs(float64(cell._originalClockRate) - float64(cell.clockRate))
	var energySpentOnReproducingDiff = math.Abs(cell._originalEnergySpentOnReproducing - cell.energySpentOnReproducing)
	var percentChanceWaitDiff = math.Abs(float64(cell._originalPercentChanceWait) - float64(cell.percentChanceWait))
	var totalDiff = growCanopyAtDiff + clockRateDiff + energyReproduceThresholdDiff + energySpentOnReproducingDiff + percentChanceWaitDiff
	//if totalDiff > SPECIES_DIVERGENCE_THRESHOLD {
	//	log("original energy threshold: %f\n", cell._originalEnergyReproduceThreshold)
	//	fmt.Printf("current energy threshold: %f\n", cell.energyReproduceThreshold)
	//	fmt.Printf("totalDiff: %f\n", totalDiff)
	//	fmt.Printf("energy spent reprod diff: %f\n", energySpentOnReproducingDiff)
	//	fmt.Printf("grew canopy at diff: %f\n", growCanopyAtDiff)
	//	fmt.Printf("energy threshold diff: %f\n", energyReproduceThresholdDiff)
	//}
	return totalDiff > SPECIES_DIVERGENCE_THRESHOLD
}

func (cell *Cell) growCanopy() {
	if cell.isDead() {
		return
	} else if cell._energy < GROWCANOPY_COST {
		//unable to reproduce, but pays cost for trying. Grim.
		cell.decreaseEnergy(GROWCANOPY_COST)
		return
		//TODO: Not sure why I need to check this condition twice, but it seems to prevent a nil reference thing
	} else if !cell.isDead() {
		cell.nextMomentSelf.canopy = true
		log(LOGTYPE_CELLEFFECT, "Cell %d grew canopy\n", cell.id)
		cell.nextMomentSelf.decreaseEnergy(GROWCANOPY_COST)
	}
}

func maintainAllCells() {
	log(LOGTYPE_MAINLOOPSINGLE, "Starting maintain\n")
	//lockAllYs("maintain cells")
	for _, cell := range currentMoment.cells {
		//lockYXRangeInclusive(cell.x, cell.y, cell.x, cell.y, "maintainer")
		cell.Maintain()
		//	unlockYXRangeInclusive(cell.x, cell.y, cell.x, cell.y, "maintainer")
	}
	//unlockAllYXs("maintain cells")
	log(LOGTYPE_MAINLOOPSINGLE, "Ending maintain\n")
}

func reapAllDeadCells() {
	log(LOGTYPE_MAINLOOPSINGLE, "Starting reap\n")
	//lockAllYs("reaper")
	var cellsToDelete = []*Cell{}
	for ci := range nextMoment.cells {
		var cell = nextMoment.cells[ci]
		if cell._energy <= 0.0 {
			cellsToDelete = append(cellsToDelete, cell)
		}
	}
	if len(cellsToDelete) > 0 {
		nextMoment.RemoveCells(cellsToDelete)
	}
	//unlockAllYXs("reaper")
	log(LOGTYPE_MAINLOOPSINGLE, "Ending reap\n")
}

func lockAllYXs(who string) {
	//lockYXRangeInclusive(0, GRID_HEIGHT-1, 0, GRID_WIDTH-1, who)
}

func lockYXRangeInclusive(startY int, endY int, startX int, endX int, who string) {
	log(LOGTYPE_DEBUGCONCURRENCY, "%s Trying to grab bulk lock\n", who)
	bulkGrabLock.Lock()
	log(LOGTYPE_DEBUGCONCURRENCY, "%s Grabbed successfully\n", who)
	for y := startY; y < endY+1; y++ {
		for x := startX; x < endX+1; x++ {
			if !isOutOfBounds(x, y, nextMoment) {
				log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to lock %d, %d\n", who, x, y)
				//nextMomentYXLocks[y][x].Lock()
			}
		}
	}
	log(LOGTYPE_DEBUGCONCURRENCY, "%s trying to release bulk lock\n", who)
	bulkGrabLock.Unlock()
}

func unlockYXRangeInclusive(startY int, endY int, startX int, endX int, who string) {
	for y := startY; y < endY+1; y++ {
		for x := startX; x < endX+1; x++ {
			if !isOutOfBounds(x, y, nextMoment) {
				log(LOGTYPE_DEBUGCONCURRENCY, "%s is going to unlock %d, %d\n", who, x, y)
				//	nextMomentYXLocks[y][x].Unlock()
			}
		}
	}
}

func unlockAllYXs(who string) {
	unlockYXRangeInclusive(0, GRID_HEIGHT-1, 0, GRID_WIDTH-1, who)
}

var speciesIDCounter = 0
var IDCounter = 0

func spontaneouslyGenerateCell() {
	//TODO: Shouldn't this be next moment?
	//TODO: Probably need to lock some stuff here
	//TODO: Cell pool should probaby zero out cell values before handing it off
	var newCell = cellPool.Borrow()
	var foundSpotYet = false
	var tries = 0
	var giveUp = false
	//TODO: Add a timeout to this
	for !foundSpotYet && !giveUp {

		var xTry = rand.Intn(GRID_WIDTH)
		var yTry = rand.Intn(GRID_HEIGHT)
		//nextMomentYXLocks[xTry][yTry].Lock()
		if !isOccupied(xTry, yTry, nextMoment) {
			foundSpotYet = true

			//TODO: This aint gonna work for some reason
			//	nextMomentLock.Lock()
			newCell.x = xTry
			newCell.y = yTry
			newCell.speciesID = speciesIDCounter
			newCell.id = IDCounter
			IDCounter += 1
			speciesIDCounter += 1
			newCell.speciesColor = getNextColor()
			newCell._energy = float64(rand.Intn(100))
			newCell._timeLeftToWait = 0

			newCell.clockRate = rand.Intn(50) + 1
			newCell.percentChanceWait = rand.Intn(40)

			newCell.energySpentOnReproducing = REPRODUCE_COST + float64(rand.Intn(20))
			newCell.energyReproduceThreshold = newCell.energySpentOnReproducing + float64(rand.Intn(120))
			newCell.canopy = false
			newCell.growCanopyAt = float64(rand.Intn(120)) + GROWCANOPY_COST
			//if rand.Intn(100) < 50 {
			//	newCell.growLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST
			//	newCell.moveChance = float64(rand.Intn(40))
			//} else {
			//		newCell.growLegsAt = float64(rand.Intn(200)) + GROWLEGS_COST + 1000
			//		newCell.moveChance = 0.0
			//	}
			newCell.growLegsAt = 9999
			newCell.legs = false

			newCell._originalMoveChance = newCell.moveChance
			newCell._originalGrowLegsAt = newCell.growLegsAt
			newCell._originalPercentChanceWait = newCell.percentChanceWait
			newCell._originalEnergyReproduceThreshold = newCell.energyReproduceThreshold
			newCell._originalEnergySpentOnReproducing = newCell.energySpentOnReproducing
			newCell._originalClockRate = newCell.clockRate
			newCell._originalgrowCanopyAt = newCell.growCanopyAt

			nextMoment.cells = append(nextMoment.cells, newCell)
			log(LOGTYPE_CELLEFFECT, "Added cell %d to next moment\n", newCell.id)
			nextMoment.cellsSpatialIndex[xTry][yTry] = newCell
		}
		//	nextMomentYXLocks[yTry][xTry].Unlock()
		tries += 1
		if tries > MAX_TRIES_TO_FIND_EMPTY_GRID_COORD {
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
		log(LOGTYPE_MAINLOOPSINGLE, "Starting shine\n")
		//TODO: This could stand to be refactored a bit
		//TODO: Changed for nextMoment to current moment, but might be wrong...

		var isDayTime = momentNum%100 <= 50
		var wg = &sync.WaitGroup{}
		for yi := range currentMoment.cellsSpatialIndex {
			//wg.Add(1)
			shineThisRow(yi, isDayTime, wg)

			//log(LOGTYPE_HIGHFREQUENCY, "Shiner touching on %d, %d \n", xi, yi)
			//unlockYXRangeInclusive(yi-1, yi+1, xi-1, xi+1, "shiner")
			//TODO: May have placed lock/unlocks here incorrectly
		}
		log(LOGTYPE_MAINLOOPSINGLE, "Ending shine\n")
		//wg.Wait()
	}

}

func shineThisRow(yi int, isDayTime bool, wg *sync.WaitGroup) {
	for xi := range currentMoment.cellsSpatialIndex[yi] {
		//var cell = currentMoment.cellsSpatialIndex[yi][xi]
		//if cell == nil {
		//	continue
		//}
		//	lockYXRangeInclusive(yi-1, yi+1, xi-1, xi+1, "shiner")
		//fmt.Printf("Proximity to middle: %d\n", int(YProximityToMiddleAsPercent(yi)*100))
		if isDayTime { //int(YProximityToMiddleAsPercent(yi)*100)

			//var shineAmountForThisSquare = SHINE_ENERGY_AMOUNT //* (float64(xi) / float64(GRID_HEIGHT)) //  GRADIENT
			var shineAmountForThisSquare = SHINE_ENERGY_AMOUNT * SHINE_FREQUENCY * (float64(xi) / GRID_WIDTH) //* float64(float64(yi)/float64(GRID_HEIGHT))
			//fmt.Printf("shine amount at %d: %f", yi, shineAmountForThisSquare)
			//var shineAmountForThisSquare = 0.0
			//if xi%10 == 0 || (xi+1)%10 == 0 || (yi%10 == 0) || ((yi+1)%10) == 0 {
			//	shineAmountForThisSquare = 0.0
			//} else {
			//	shineAmountForThisSquare = SHINE_ENERGY_AMOUNT
			//}
			newShineMethod(xi, yi, shineAmountForThisSquare)
			//oldShineMethod(cell, shineAmountForThisSquare)
		} else {
			//No sun at night
		}
	}
	//wg.Done()
}

func oldShineMethod(cell *Cell, shineAmountForThisSquare float64) {
	var surroundingCells = getSurroundingCells(cell.x, cell.y, currentMoment)
	var surroundingCellsWithCanopy = make([]*Cell, 0, 9)
	for _, surroundingCell := range surroundingCells {
		if surroundingCell.canopy == true {
			surroundingCellsWithCanopy = append(surroundingCellsWithCanopy, surroundingCell)
		}
	}
	var surroundingCellsWithCanopyAndMe = append(surroundingCellsWithCanopy, cell)
	var energyToEachCell = shineAmountForThisSquare / float64(len(surroundingCellsWithCanopyAndMe))
	for _, cellToReceiveEnergy := range surroundingCellsWithCanopyAndMe {
		cellToReceiveEnergy.increaseEnergy(energyToEachCell)
	}
}

func newShineMethod(x int, y int, shineAmountForThisSquare float64) {
	//TODO: Next up, recycle these slices?
	//TODO: Just making the array is faster than using pool. Surprising
	var surroundingCellsWithCanopiesAndMe = &[SURROUNDINGS_SIZE]*Cell{} //surroundingsPool.Borrow()
	var numSurrounders = 0
	var cell = currentMoment.cellsSpatialIndex[y][x]
	if cell != nil {
		surroundingCellsWithCanopiesAndMe[0] = cell
		numSurrounders += 1
	}
	//lockYXRangeInclusive(y-1, y+1, x-1, x+1, "shiner")
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = x + relativeX
			var yTry = y + relativeY
			if relativeX == 0 && relativeY == 0 {
				continue
			}
			if !isOutOfBounds(xTry, yTry, currentMoment) && isOccupied(xTry, yTry, currentMoment) && currentMoment.cellsSpatialIndex[xTry][yTry].canopy == true {
				var surroundingCell = currentMoment.cellsSpatialIndex[xTry][yTry]
				surroundingCellsWithCanopiesAndMe[numSurrounders] = surroundingCell
				numSurrounders += 1
			}
		}
	}
	//unlockYXRangeInclusive(y-1, y+1, x-1, x+1, "shiner")
	var energyToEachCell = shineAmountForThisSquare / float64(numSurrounders)
	for i := 0; i < numSurrounders; i++ {
		surroundingCellsWithCanopiesAndMe[i].increaseEnergy(energyToEachCell)
	}

	//surroundingsPool.Return(surroundingCellsWithCanopiesAndMe)
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
	surroundingCells = make([]*Cell, 0, 9)
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = x + relativeX
			var yTry = y + relativeY
			//Inlined from !outofBounds and isOccupied
			if !(xTry < 0 || xTry > GRID_WIDTH-1 || yTry < 0 || yTry > GRID_HEIGHT-1) && moment.cellsSpatialIndex[x][y] != nil {
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
	return x < 0 || x > GRID_WIDTH-1 || y < 0 || y > GRID_HEIGHT-1
}

func isXOutOfBounds(x int, moment *Moment) bool {
	return x < 0 || x > GRID_WIDTH-1
}

func isYOutOfBounds(y int, moment *Moment) bool {
	return y < 0 || y > GRID_HEIGHT-1
}
