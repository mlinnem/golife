// +build tracing

package lib

import "fmt"

func Log(logType int, message string, params ...interface{}) {
	if containsInt(LOGTYPES_ENABLED, logType) {
		fmt.Printf(message, params...)
	}
}

func LogIfTraced(cell *Cell, logType int, message string, params ...interface{}) {
	if DEBUG && ((cell != nil && CELLEFFECT_ONLY_IF_TRACED == false) || (TracedCell != nil && cell.ID == TracedCell.ID)) {
		Log(logType, "TRACED: "+message, params...)
		return true
	}
	return false
}
