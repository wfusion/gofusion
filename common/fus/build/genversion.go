package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	gitCommit, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	gitTag, _ := exec.Command("git", "describe", "--tags").Output()

	versionFileContent := fmt.Sprintf(`
package main

var (
	gitCommit = "%s"
	gitTag    = "%s"
)
`, strings.TrimSpace(string(gitCommit)), strings.TrimSpace(string(gitTag)))

	f, err := os.Create("version.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.WriteString(versionFileContent)
	if err != nil {
		panic(err)
	}
}
