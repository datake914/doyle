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
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	tmp, err := ioutil.TempDir(filepath.Dir(exe), "tmp")
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	conf.tmpDir = &tmp

	// Create a source server client.
	sourceClient, err := NewSshClient(conf.sourceServer)
	if err != nil {
		fmt.Println(err)
	}
	// Create a target server client.
	targetClient, err := NewSshClient(conf.targetServer)
	if err != nil {
		fmt.Println(err)
	}
	clients := [2]Client{sourceClient, targetClient}

	// Create target file list.
	c := make(chan string)
	for _, client := range clients {
		go func(c chan string, client Client) {
			result, err := FindToFile(client, conf)
			if err != nil {
				fmt.Printf("%+v\n", err)
				os.Exit(1)
			}
			c <- result.StdoutPath
		}(c, client)
	}
	sourcePath, targetPath := <-c, <-c
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		fmt.Printf("File %s could not read: %v\n", sourcePath, err)
		os.Exit(1)
	}
	defer sourceFile.Close()
	targetFile, err := os.Open(targetPath)
	if err != nil {
		fmt.Printf("File %s could not read: %v\n", targetPath, err)
		os.Exit(1)
	}
	defer targetFile.Close()

	sourceScanner, targetScanner := bufio.NewScanner(sourceFile), bufio.NewScanner(targetFile)
	sourceNext, targetNext := sourceScanner.Scan(), targetScanner.Scan()
	scstat, tcstat, sccat, tccat := make(chan *Result), make(chan *Result), make(chan *Result), make(chan *Result)
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
				go stat(scstat, sourceClient, sourceFileName)
				go stat(tcstat, targetClient, targetFileName)
				go catMd5sum(sccat, sourceClient, sourceFileName)
				go catMd5sum(tccat, targetClient, targetFileName)

				sourceStatResult, targetStatResult, sourceCatMd5sumResult, targetCatMd5sumResult := <-scstat, <-tcstat, <-sccat, <-tccat
				if sourceStatResult.ExitStatus != 999 && targetStatResult.ExitStatus != 999 && sourceCatMd5sumResult.ExitStatus != 999 && targetCatMd5sumResult.ExitStatus != 999 {
					if sourceStat, targetStat, sourceContent, targetContent := sourceStatResult.Stdout, targetStatResult.Stdout, sourceCatMd5sumResult.Stdout, targetCatMd5sumResult.Stdout; sourceStat != targetStat || sourceContent != targetContent {
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
				} else {
					fmt.Printf("[ERROR] %s diff failed. Try again.", sourceFileName)
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
	if err := os.RemoveAll(tmp); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func stat(c chan *Result, client Client, fileName string) {
	result, err := Stat(client, fileName)
	if err != nil {
		fmt.Printf("%+v\n", err)
		// os.Exit(1)
	}
	c <- result
}

func catMd5sum(c chan *Result, client Client, fileName string) {
	result, err := CatMd5sum(client, fileName)
	if err != nil {
		fmt.Printf("%+v\n", err)
		// os.Exit(1)
	}
	c <- result
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
