package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
)

func IsBoxnameValid(boxname string) bool {
	return regexp.MustCompile(`^[\w-]{1,}$`).MatchString(boxname)
}

func AskConfirmation(msg string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/N]: ", msg)

	response, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y"
}

func PrintCommand(cmd ...string) {
	var isprepreviousoption bool
	var ispreviousoption bool

	for i, c := range cmd {
		isoption := strings.HasPrefix(c, "--")

		switch {
		case isprepreviousoption && !ispreviousoption && !isoption:
			fmt.Print("\n")
		case ispreviousoption && !isoption:
			fmt.Print(" ")
		case ispreviousoption && isoption:
			fmt.Print("\n")
		case isoption:
			fmt.Print("\n")
		case !ispreviousoption && !isoption && i != 0:
			fmt.Print(" ")
		}

		fmt.Print(c)

		isprepreviousoption = ispreviousoption
		ispreviousoption = isoption
	}

	fmt.Println()
}

func TrimDoc(s string) string {
	n := math.MaxInt
	for line := range strings.SplitSeq(s, "\n") {
		var nl int

		for _, r := range []byte(line) {
			if r != ' ' && r != '\t' {
				break
			}
			nl++
		}

		if nl != 0 && nl != len(line) {
			n = min(n, nl)
		}
	}

	if n == math.MaxInt {
		n = 0
	}

	var b strings.Builder
	for line := range strings.SplitSeq(s, "\n") {
		if len(line) < n {
			b.WriteString("\n")
			continue
		}

		b.WriteString(line[n:])
		b.WriteString("\n")
	}

	return b.String()
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return false
}
