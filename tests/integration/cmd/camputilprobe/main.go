package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Obedience-Corp/obey-shared/camputil"
)

func main() {
	startDir := ""
	if len(os.Args) > 1 {
		startDir = os.Args[1]
	}

	root, err := camputil.FindCampaignRoot(context.Background(), startDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(root)
}
