package main

import (
	"testing"
)

func TestGetNodeMinor(t *testing.T) {
	var tMinor typeNodeMinor
	var err error

	i := 0
	for ; i < len(tMinor.minorsInUse); i++ {
		n, err := tMinor.getNodeMinor()
		if n != i || err != nil {
			t.FailNow()
			return
		}
	}

	// getNodeMinor should return error if more than maxMinor are returned
	_, err = tMinor.getNodeMinor()
	if err == nil {
		t.FailNow()
	}
}

func TestPutNodeMinor(t *testing.T) {

	var tMinor typeNodeMinor
	var err error

	// if there is no minor nuber in use, it shouldn't accept any put operation
	n := 0
	i := 0
	for ; i < len(tMinor.minorsInUse); i++ {
		err = tMinor.putNodeMinor(i)
		if err == nil {
			t.FailNow()
			return
		}
	}

	// get out all minor numbers and put them back
	for ; i < len(tMinor.minorsInUse); i++ {
		n, err = tMinor.getNodeMinor()
		if n != i || err != nil {
			t.FailNow()
			return
		}
	}
	for ; i < len(tMinor.minorsInUse); i++ {
		err = tMinor.putNodeMinor(i)
		if err != nil {
			t.FailNow()
			return
		}
	}
}

func TestGetOncePutTwice(t *testing.T) {
	var tMinor typeNodeMinor
	var err error

	i := 0
	// get once and put twice
	_, err = tMinor.getNodeMinor()
	if err != nil {
		t.FailNow()
		return
	}
	err = tMinor.putNodeMinor(i)
	if err != nil {
		t.Error()
		t.FailNow()
		return
	}
	err = tMinor.putNodeMinor(i)
	if err == nil {
		t.FailNow()
		return
	}
}

func TestPutOneGetOneBack(t *testing.T) {
	var tMinor typeNodeMinor
	var err error
	const testIndex = 100

	n := 0
	i := 0
	// get out all minor numbers and put them back
	for ; i < len(tMinor.minorsInUse); i++ {
		n, err = tMinor.getNodeMinor()
		if n != i || err != nil {
			t.FailNow()
			return
		}
	}

	err = tMinor.putNodeMinor(testIndex)
	if err != nil {
		t.Error()
		t.FailNow()
		return
	}

	n, err = tMinor.getNodeMinor()
	if err != nil || n != testIndex {
		t.FailNow()
		return
	}
}
