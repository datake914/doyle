package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
	// Parse command line arguments.
	conf := parse()
	// Create a temporary directory.
	conf.tmpDir = createTempDir()

	// Create a source server client.
	sourceClient := createClient(conf.sourceServer)
	// Create a target server client.
	targetClient := createClient(conf.targetServer)

	// Create check file list.
	sc, tc := make(chan string), make(chan string)
	go find(sc, sourceClient, conf)
	go find(tc, targetClient, conf)
	sourceFile, targetFile := openFile(<-sc), openFile(<-tc)
	defer sourceFile.Close()
	defer targetFile.Close()

	sourceScanner, targetScanner := bufio.NewScanner(sourceFile), bufio.NewScanner(targetFile)
	sourceNext, targetNext := sourceScanner.Scan(), targetScanner.Scan()
	for {
		sourceFileName, targetFileName := sourceScanner.Text(), targetScanner.Text()
		if !sourceNext && !targetNext {
			// When both files are read to the end.
			break
		} else if !sourceNext {
			// When only source file is read to the end.
			fmt.Println("D " + targetFileName)
			targetNext = targetScanner.Scan()
		} else if !targetNext {
			// When only target file is read to the end.
			fmt.Println("A " + sourceFileName)
			sourceNext = sourceScanner.Scan()
		} else {
			switch strings.Compare(sourceFileName, targetFileName) {
			// When the source file name is equal to the target file name.
			case 0:
				// Get file stat.
				go stat(sc, sourceClient, sourceFileName)
				go stat(tc, targetClient, targetFileName)
				sourceStat, targetStat := <-sc, <-tc
				// Get file content.
				go catMd5sum(sc, sourceClient, sourceFileName)
				go catMd5sum(tc, targetClient, targetFileName)
				sourceContent, targetContent := <-sc, <-tc

				if sourceStat != targetStat || sourceContent != targetContent {
					fmt.Println("M " + sourceFileName)
					if *conf.detail {
						if sourceStat != targetStat {
							diff(sourceStat, targetStat)
						} else {
							diff(sourceContent, targetContent)
						}
					}
				} else {
					fmt.Println("  " + sourceFileName)
				}
				sourceNext = sourceScanner.Scan()
				targetNext = targetScanner.Scan()
			// When the target file does not exist in the source server.
			case 1:
				fmt.Println("D " + targetFileName)
				targetNext = targetScanner.Scan()
			// When the source file does not exist in the target server.
			case -1:
				fmt.Println("A " + sourceFileName)
				sourceNext = sourceScanner.Scan()
			}
		}
	}
	// Delete a temporary directory.
	removeTempDir(conf.tmpDir)
}

func createTempDir() string {
	exe, err := os.Executable()
	if err != nil {
		handleErr(err)
	}
	tmp, err := ioutil.TempDir(filepath.Dir(exe), "tmp")
	if err != nil {
		handleErr(err)
	}
	return tmp
}

func removeTempDir(path string) {
	if err := os.RemoveAll(path); err != nil {
		handleErr(err)
	}
}

func createClient(conf *ServerConfig) Client {
	switch *conf.Host {
	case "localhost":
		fmt.Println("localhost target is not supported.")
	default:
		client, err := NewSshClient(conf)
		if err != nil {
			handleErr(fmt.Errorf("client creation failed: %s", err))
		}
		return client
	}
	return nil
}

func openFile(path string) *os.File {
	file, err := os.Open(path)
	if err != nil {
		handleErr(fmt.Errorf("File %s could not read: %v", path, err))
	}
	return file
}

func find(c chan string, client Client, conf *config) {
	result, err := FindToFile(client, conf)
	if err != nil {
		handleErr(err)
	}
	c <- result.Stdout
}

func stat(c chan string, client Client, fileName string) {
	result, err := Stat(client, fileName)
	if err != nil {
		handleErr(err)
	}
	c <- result.Stdout
}

func catMd5sum(c chan string, client Client, fileName string) {
	result, err := CatMd5sum(client, fileName)
	if err != nil {
		handleErr(err)
	}
	c <- result.Stdout
}

func diff(source, target string) {
	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToChars(source, target)
	diffs := dmp.DiffCharsToLines(dmp.DiffMain(a, b, false), c)
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			fmt.Println("> " + diff.Text)
		case diffmatchpatch.DiffInsert:
			fmt.Println("< " + diff.Text)
		}
	}
}

func handleErr(err error) {
	fmt.Printf("%+v\n", err)
	os.Exit(1)
}
