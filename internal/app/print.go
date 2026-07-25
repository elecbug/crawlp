package app

import (
	"bufio"
	"fmt"
	"os"

	"github.com/elecbug/crawlp/internal/config"
)

func PrintBanner() {
	fmt.Println("========================================")
	fmt.Println(config.APP_NAME)
	fmt.Println("========================================")
}

func PrintError(err error) {
	fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
}

func WaitForEnter() {
	fmt.Println()
	fmt.Print("Press Enter to exit...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
