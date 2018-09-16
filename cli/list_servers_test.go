package cli

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestListDirContents(t *testing.T) {
	mkdir := exec.Command("sh", "-c", "mkdir testfolder testfolder/hello testfolder/world")
	mkdir.Dir = "."

	rm := exec.Command("sh", "-c", "rm -rf testfolder")
	rm.Dir = "."
	defer rm.Run()

	_, err := mkdir.Output()
	if err == nil {
		fmt.Println(err)
		t.Fail()
	}

	contents, err := ListDirContents("testfolder")

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if contents[0] != "hello" || contents[1] != "world" {
		fmt.Println("Dirs hello and world not present")
		t.Fail()
	}
	if len(contents) != 2 {
		fmt.Printf("Expected 2 directories, got %d\n", len(contents))
		for i := 0; i < len(contents); i++ {
			fmt.Printf("%s", contents[i])
			fmt.Printf("\n")
		}
		t.Fail()
	}
}
