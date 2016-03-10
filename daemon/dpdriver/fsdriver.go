package dpdriver

import (
//"fmt"
)

type fsdriver struct {
}

func (fs *fsdriver) GetDestFileName(dpconn, itemlocation, filename string) (destfilename, tmpdir, tmpfile string) {
	destfilename = dpconn + "/" + itemlocation + "/" + filename
	tmpdir = dpconn + "/" + itemlocation + "/tmp"
	tmpfile = tmpdir + "/" + filename
	return
}

func (fs *fsdriver) StoreFile(status, filename, dpconn, dp, itemlocation, destfile string) string {
	return status
}
func init() {
	//fmt.Println("fs")
	register("file", &fsdriver{})
}
