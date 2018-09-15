package cli

import "testing"

func TestIsFlag(t *testing.T) {
	flagStrings := []string{"-d", "--delete"}
	nonFlagStrings := []string{"staging", "my-branch", "-dumb-branch", ""}
	for i := 0; i < len(flagStrings); i++ {
		if !isFlag(flagStrings[i]) {
			t.Fail()
		}
	}

	for i := 0; i < len(nonFlagStrings); i++ {
		if isFlag(nonFlagStrings[i]) {
			t.Fail()
		}
	}
}
