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

type Summery struct {
	totalCount int
	addCount   int
	delCount   int
	modCount   int
}

func main() {
	if err := execute(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func execute() (err error) {
	// Parse command line arguments.
	conf := parse()
	// Create a temporary directory.
	conf.tmpDir, err = createTempDir()
	if err != nil {
		return
	}
	defer removeTempDir(conf.tmpDir)

	// Create a source server client.
	sourceClient, err := createClient(conf.sourceServer)
	if err != nil {
		return
	}
	// Create a target server client.
	targetClient, err := createClient(conf.targetServer)
	if err != nil {
		return
	}

	// Create check file lists.
	sc, tc := make(chan string), make(chan string)
	go find(sc, sourceClient, conf)
	go find(tc, targetClient, conf)
	h, f := <-sc, <-tc
	sourceFile, err := openFile(h)
	if err != nil {
		return
	}
	targetFile, err := openFile(f)
	if err != nil {
		return
	}
	defer sourceFile.Close()
	defer targetFile.Close()

	// Scan the file lists and check diff.
	sourceScanner, targetScanner := bufio.NewScanner(sourceFile), bufio.NewScanner(targetFile)
	sourceNext, targetNext := sourceScanner.Scan(), targetScanner.Scan()
	fmt.Println("#### Diff Report ####")
	summery := new(Summery)
	for {
		summery.totalCount += 1
		sourceFileName, targetFileName := sourceScanner.Text(), targetScanner.Text()
		if !sourceNext && !targetNext {
			// When both files are read to the end.
			break
		} else if !sourceNext {
			// When only source file is read to the end.
			fmt.Println("A " + targetFileName)
			summery.addCount += 1
			targetNext = targetScanner.Scan()
		} else if !targetNext {
			// When only target file is read to the end.
			fmt.Println("D " + sourceFileName)
			summery.delCount += 1
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
					summery.modCount += 1
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
				fmt.Println("A " + targetFileName)
				summery.addCount += 1
				targetNext = targetScanner.Scan()
			// When the source file does not exist in the target server.
			case -1:
				fmt.Println("D " + sourceFileName)
				summery.delCount += 1
				sourceNext = sourceScanner.Scan()
			}
		}
	}
	fmt.Println("#### Diff Summery ####")
	fmt.Printf("Source Host: %s\n", *conf.sourceServer.Host)
	fmt.Printf("Target Host: %s\n", *conf.targetServer.Host)
	fmt.Printf("Check: %s\n", strings.Join(*conf.targets, " "))
	fmt.Printf("Total: %d files, Diffs: %d (Add: %d, Del: %d, Mod: %d)",
		summery.totalCount, summery.addCount+summery.delCount+summery.modCount,
		summery.addCount, summery.delCount, summery.modCount)
	return
}

func createTempDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	tmp, err := ioutil.TempDir(filepath.Dir(exe), "tmp")
	if err != nil {
		return "", err
	}
	return tmp, nil
}

func removeTempDir(path string) error {
	return os.RemoveAll(path)
}

func createClient(conf *ServerConfig) (Client, error) {
	switch *conf.Host {
	case "localhost":
		fmt.Println("localhost target is not supported.")
	default:
		client, err := NewSshClient(conf)
		if err != nil {
			return nil, fmt.Errorf("client creation failed: %s", err)
		}
		return client, nil
	}
	return nil, nil
}

func openFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("File %s could not read: %v", path, err)
	}
	return file, nil
}

func find(c chan string, client Client, conf *config) error {
	result, err := FindToFile(client, conf)
	if err != nil {
		return err
	}
	c <- result.StdoutPath
	return nil
}

func stat(c chan string, client Client, fileName string) error {
	result, err := Stat(client, fileName)
	if err != nil {
		return err
	}
	c <- result.Stdout
	return nil
}

func catMd5sum(c chan string, client Client, fileName string) error {
	result, err := CatMd5sum(client, fileName)
	if err != nil {
		return err
	}
	c <- result.Stdout
	return nil
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
