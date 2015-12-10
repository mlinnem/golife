package lib

import "sync"

var OldMoment *Moment
var MomentBeingCleaned *Moment
var momentReadyToBeNext *Moment
var CurrentMoment *Moment
var NextMoment *Moment

type Moment struct {
	MomentNum         int
	Cells             []*Cell
	CellsSpatialIndex [GRID_WIDTH][GRID_HEIGHT]*Cell
}

//TODO: Maybe do this in tandem with goroutine in future
func (moment *Moment) Clean(wg *sync.WaitGroup) {
	//NextMomentLock.Lock()
	//for ci := range moment.Cells {
	//moment.Cells[ci] = nil
	//}
	moment.Cells = []*Cell{}
	//moment.CellsSpatialIndex = [][]*Cell{{}}
	//var internalwg sync.WaitGroup
	//TODO: Put log back in when you can make log its own module thingie
	//Log(LOGTYPE_MAINLOOPSINGLE, "You made it to right before allocating the big cell thing\n")
	//TODO: Letting garbage collector take care of cleaning rather than manual for now
	moment.CellsSpatialIndex = [GRID_WIDTH][GRID_HEIGHT]*Cell{}
	// for yi := range moment.CellsSpatialIndex {
	// 	//internalwg.Add(1)
	// 	moment.CleanRow(yi)
	// }
	// //internalwg.Wait()
	wg.Done()
	//NextMomentLock.Unlock()
}

func (moment *Moment) CleanRow(yi int) {
	for xi := range moment.CellsSpatialIndex[yi] {
		moment.CellsSpatialIndex[xi][yi] = nil
	}
	//wg.Done()
}

func (moment *Moment) IsOccupied(x int, y int) bool {
	if moment.IsOutOfBounds(x, y) {
		return true
	}
	return moment.CellsSpatialIndex[x][y] != nil
}

func (moment *Moment) ReturnCellsToPool() {
	for ci := range moment.Cells {
		var cellToReturn = moment.Cells[ci]
		CellPool.Return(cellToReturn)
	}
}

func (moment *Moment) IsOutOfBounds(x int, y int) bool {
	return x < 0 || x > GRID_WIDTH-1 || y < 0 || y > GRID_HEIGHT-1
}

func (moment *Moment) isXOutOfBounds(x int) bool {
	return x < 0 || x > GRID_WIDTH-1
}

func (moment *Moment) isYOutOfBounds(y int) bool {
	return y < 0 || y > GRID_HEIGHT-1
}
