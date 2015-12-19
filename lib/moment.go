package lib

import "sync"

var OldMoment *Moment
var MomentBeingCleaned *Moment
var momentReadyToBeNext *Moment
var CurrentMoment *Moment
var NextMoment *Moment

type Moment struct {
	MomentNum                int
	Cells                    []*Cell
	SpatialIndexSolid        [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	SpatialIndexSurfaceCover [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	//SpatialIndexSurfaceCoverNonSolid     [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	//SpatialIndexSurfaceCover[GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
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
	//moment.SpatialIndexSurfaceCover = [][]*Cell{{}}
	//var internalwg sync.WaitGroup
	//TODO: Put log back in when you can make log its own module thingie
	//Log(LOGTYPE_MAINLOOPSINGLE, "You made it to right before allocating the big cell thing\n")
	//TODO: Letting garbage collector take care of cleaning rather than manual for now
	//	moment.SpatialIndexSurfaceCoverFilling = [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell{}
	moment.SpatialIndexSolid = [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell{}
	moment.SpatialIndexSurfaceCover = [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell{}

	// for yi := range moment.SpatialIndexSurfaceCover {
	// 	//internalwg.Add(1)
	// 	moment.CleanRow(yi)
	// }
	// //internalwg.Wait()
	wg.Done()
	//NextMomentLock.Unlock()
}

func (moment *Moment) AddCellToSpatialIndex(cell *Cell) {
	//Cover-Surface == if not legs, Cover-Ground at z + height
	//Non-Solid == 1 around for canopy on same z level, and 1 in current x,y,z
	//Solid == 1 for each height, starting with Z level.
	//fmt.Printf("%d %d %d\n", cell.Z+i, cell.Y, cell.X)
	if cell.Height >= 1 {
		for i := 0; i < cell.Height; i++ {
			moment.SpatialIndexSolid[cell.Z+i][cell.Y][cell.X] = cell
		}
	}
	moment.SpatialIndexSurfaceCover[cell.Z+cell.Height][cell.Y][cell.X] = cell
}

func (moment *Moment) RemoveCellFromSpatialIndex(cell *Cell) {
	if cell.Height >= 1 {
		for i := 0; i < cell.Height; i++ {
			moment.SpatialIndexSolid[cell.Z+i][cell.Y][cell.X] = nil
		}
	}
	moment.SpatialIndexSurfaceCover[cell.Z+cell.Height][cell.Y][cell.X] = nil
}

func (moment *Moment) IsSolidOrCovered(x int, y int, z int) bool {
	if moment.IsOutOfBounds(x, y, z) {
		return true
	}
	//TODO: Might want to just out and out make it 3D at some point
	return moment.IsSolid(x, y, z) || moment.IsCovered(x, y, z)
}

func (moment *Moment) IsCovered(x, y, z int) bool {
	return moment.SpatialIndexSurfaceCover[z][y][x] != nil
}

func (moment *Moment) IsSolid(x, y, z int) bool {
	return moment.SpatialIndexSolid[z][y][x] != nil
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
			//Inlined from !outofBounds and IsSolidOrCovered
			if !(xTry < 0 || xTry > GRID_WIDTH-1 || yTry < 0 || yTry > GRID_HEIGHT-1 || z < 0 || z > GRID_DEPTH) && moment.SpatialIndexSurfaceCover[z][y][x] != nil {
				var cell = moment.SpatialIndexSurfaceCover[z][yTry][xTry]
				surroundingCells = append(surroundingCells, cell)
			}
		}
	}
	return surroundingCells
}
