package main

import (
	"fmt"
)

const maxMinor = 255

// A simple implementation to organize minor number assignment for one device type
type typeNodeMinor struct {
	// index of the array represents each minor number from 0 to 255.
	// value false: the minor number is not in use
	// value true: the minor number is in use now
	minorsInUse [maxMinor + 1]bool
}

func (nMinor *typeNodeMinor) getNodeMinor() (int, error) {
	for i, inUse := range nMinor.minorsInUse {
		if inUse {
			continue
		}

		nMinor.minorsInUse[i] = true
		return i, nil
	}

	return -1, fmt.Errorf("there is no idle minor number for this type")
}

func (nMinor *typeNodeMinor) putNodeMinor(minorNum int) error {
	if nMinor.minorsInUse[minorNum] == false {
		return fmt.Errorf("Minor number %d is not in use", minorNum)
	}

	nMinor.minorsInUse[minorNum] = false
	return nil
}
