package task

import (
	"regexp"
	"fmt"
)

//[^\da-zA-Z-]

var (
	normalNameQualifier = regexp.MustCompile("[^\\w-]")
	snapshotNameQualifier = regexp.MustCompile("[^\\w-.]")
	imageNameQualifier = regexp.MustCompile("[^\\w-.]")
)

func QualifyNormalName(input string) (err error){
	var matched = normalNameQualifier.FindStringSubmatch(input)
	if 0 != len(matched){
		err = fmt.Errorf("illegal char '%s' (only '0~9a~Z_-' allowed)", matched[0])
		return err
	}
	return nil
}

func QualifyImageName(snapshot string) (err error){
	var matched = imageNameQualifier.FindStringSubmatch(snapshot)
	if 0 != len(matched){
		err = fmt.Errorf("illegal char '%s' (only '0~9a~Z_-.' allowed)", matched[0])
		return err
	}
	return nil
}

func QualifySnapshotName(snapshot string) (err error){
	var matched = snapshotNameQualifier.FindStringSubmatch(snapshot)
	if 0 != len(matched){
		err = fmt.Errorf("illegal char '%s' (only '0~9a~Z_-.' allowed)", matched[0])
		return err
	}
	return nil
}
