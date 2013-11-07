package main

import (
	// "calf/image"
	"calf/storage"
	// "encoding/json"
	"fmt"
	// "io/ioutil"
	// "os"
	// "path"
)

var cmdImport = &Command{
	UsageLine: "import [filename]",
	Short:     "import data from imsto old version or file",
	Long: `
import from a image file
`,
}

func init() {
	cmdImport.Run = runImport
}

func runImport(args []string) bool {
	if len(args) == 0 {
		fmt.Println("nothing")
		return false
	} else {
		fmt.Println(args[0])
	}

	section := ""
	var entry *storage.Entry
	entry, err := storage.StoredFile(args[0], section)

	if err != nil {
		fmt.Println(err)
		return false
	}

	fmt.Println("entry stored: %s\n", entry.Id)

	// var mw storage.MetaWrapper
	// mw = storage.NewMetaWrapper("")
	// // fmt.Println("mw", mw)

	// err = mw.Store(entry)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return false
	// }

	return true
}
