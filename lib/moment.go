package lib

import "sync"

var OldMoment *Moment
var MomentBeingCleaned *Moment
var momentReadyToBeNext *Moment
var CurrentMoment *Moment
var NextMoment *Moment

type Moment struct {
	MomentNum int
	Cells     []*Cell
	//	CellsSpatialIndexSolid        [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	//	CellsSpatialIndexCoverSurface [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	//CellsSpatialIndexNonSolid     [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	CellsSpatialIndex [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	//	atmosphericMaterial float64
}

//func (moment *Moment) SetTotalAtmosphericMaterial(amt float64) {
//	moment.atmosphericMaterial = amt
//}

//func (moment *Moment) GetTotalAtmosphericMaterial(amt float64) float64 {
//	return moment.atmosphericMaterial
//}

//func (moment *Moment) ReleaseMaterialToAtmosphere(amt float64) {
//	moment.atmosphericMaterial += amt
//}

//TODO: Kind of weird to always return exactly what is available
//func (moment *Moment) TakeMaterialFromAtmosphere(amt float64) float64 {
//	moment.atmosphericMaterial -= amt
//	return amt
//}

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
	//	moment.CellsSpatialIndexFilling = [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell{}
	moment.CellsSpatialIndex = [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell{}
	// for yi := range moment.CellsSpatialIndex {
	// 	//internalwg.Add(1)
	// 	moment.CleanRow(yi)
	// }
	// //internalwg.Wait()
	wg.Done()
	//NextMomentLock.Unlock()
}

func (moment *Moment) AddCellToSpatialIndex(cell *Cell) {
	for i := 0; i < cell.Height+1; i++ {
		//Cover-Surface == if not legs, Cover-Ground at z + height
		//Non-Solid == 1 around for canopy on same z level, and 1 in current x,y,z
		//Solid == 1 for each height, starting with Z level.
		//fmt.Printf("%d %d %d\n", cell.Z+i, cell.Y, cell.X)
		moment.CellsSpatialIndex[cell.Z+i][cell.Y][cell.X] = cell
	}
}

func (moment *Moment) RemoveCellFromSpatialIndex(cell *Cell) {
	for i := 0; i < cell.Height+1; i++ {
		moment.CellsSpatialIndex[cell.Z+i][cell.Y][cell.X] = nil
	}
}

func (moment *Moment) CleanRow(yi int, zi int) {
	for xi := range moment.CellsSpatialIndex[zi][yi] {
		moment.CellsSpatialIndex[zi][yi][xi] = nil
	}
	//wg.Done()
}

func (moment *Moment) IsOccupied(x int, y int, z int) bool {
	if moment.IsOutOfBounds(x, y, z) {
		return true
	}
	//TODO: Might want to just out and out make it 3D at some point
	return moment.CellsSpatialIndex[z][y][x] != nil
}

func (moment *Moment) ReturnCellsToPool() {
	for ci := range moment.Cells {
		var cellToReturn = moment.Cells[ci]
		CellPool.Return(cellToReturn)
	}
}

func (moment *Moment) IsOutOfBounds(x int, y int, z int) bool {
	return x < 0 || x > GRID_WIDTH-1 || y < 0 || y > GRID_HEIGHT-1 || z < 0 || z > GRID_DEPTH-1
}

//In the plane of z
func (moment *Moment) getSurroundingCells(x int, y int, z int) []*Cell {
	var surroundingCells []*Cell
	surroundingCells = make([]*Cell, 0, 9)
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = x + relativeX
			var yTry = y + relativeY
			//Inlined from !outofBounds and IsOccupied
			if !(xTry < 0 || xTry > GRID_WIDTH-1 || yTry < 0 || yTry > GRID_HEIGHT-1 || z < 0 || z > GRID_DEPTH) && moment.CellsSpatialIndex[z][y][x] != nil {
				var cell = moment.CellsSpatialIndex[z][yTry][xTry]
				surroundingCells = append(surroundingCells, cell)
			}
		}
	}
	return surroundingCells
}
