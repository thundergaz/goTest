package goUnitTest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func scanFold() {
	filepath.Walk("./",
		func(path string, f os.FileInfo, err error) error {
			if strings.HasSuffix(path, ".go") {
				fmt.Println("golang file:", path)
			}
			if f == nil {
				return err
			}
			return nil
		})
}