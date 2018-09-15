package cli

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"testing"
)

func TestEnsureBranchExists(t *testing.T) {
	ctx := context.Background()
	client := github.NewClient(nil)

	if err := EnsureBranchExists(ctx, client, "master"); err != nil {
		fmt.Println("Did not find branch master")
		t.Fail()
	}

	if err := EnsureBranchExists(ctx, client, "non_existent_branch"); err == nil {
		fmt.Println("Found branch non_existent_branch")
		t.Fail()
	}
}
