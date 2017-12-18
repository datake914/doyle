package main

import (
	"fmt"
	"io/ioutil"
)

func Find(client Client, conf *config) (*Result, error) {
	return client.exec(createFindCommnad(conf), true)
}

func FindToFile(client Client, conf *config) (*Result, error) {
	stdOut, err := ioutil.TempFile(*conf.tmpDir, "find")
	if err != nil {
		return nil, fmt.Errorf("Failed to create temporary file: %s", err)
	}
	defer stdOut.Close()
	result, err := client.execWithPipe(createFindCommnad(conf), true, &stdConfig{Stdout: stdOut})
	if result != nil {
		result.StdoutPath = stdOut.Name()
	}
	return result, err
}

func Exists(client Client, target string) (*Result, error) {
	return client.exec(createExistsCommand(target), true)
}

func Stat(client Client, target string) (*Result, error) {
	return client.exec(createStatCommand(target), true)
}

func CatMd5sum(client Client, target string) (*Result, error) {
	return client.exec(createMd5sumCommand(target), true)
}

func createFindCommnad(conf *config) string {
	// Get find targets.
	targets := ""
	for _, v := range *conf.targets {
		targets += escapeOption(v) + " "
	}
	// Get exclude targets.
	excludes := ""
	for _, v := range *conf.excludes {
		excludes += "-not -path " + escapeOption(v) + " "
	}
	return "find " + targets + excludes + "| sort"
}

func createExistsCommand(target string) string {
	return "test -e " + escapeOption(target)
}

func createStatCommand(target string) string {
	return "stat -c " + escapeOption("%A %g %G %u %U") + " " + escapeOption(target)
}

func createMd5sumCommand(target string) string {
	return "md5sum " + escapeOption(target)
}

func createCatMd5sumCommand(target string) string {
	target = escapeOption(target)
	return "if (file -b " + target + " | grep text > /dev/null 2>&1 && test `wc -c " + target + " | awk '{print $1}' -gt 512000` ); then (cat " + target + ") else (md5sum " + target + ") fi"
}

func escapeOption(str string) string {
	return "'" + str + "'"
}
