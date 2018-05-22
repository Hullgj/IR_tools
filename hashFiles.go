package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Globals
var ProgramName, ProgramSeverity string = "", ""
var HashArray = make(map[string]string) // hash: filename

func checkErr(err error) {
	if err != nil {
		var mylog = log.New(os.Stderr, "app: ", log.LstdFlags|log.Lshortfile)
		mylog.Fatal(err)
	}
}

func makeResultsFile(filename *string) *os.File {
	fileResults, err := os.OpenFile(*filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	checkErr(err)
	// add the header for the CSV file
	fileResults.WriteString("ioctype,iocname,severity,data,comment\r\n")
	return fileResults
}

func updateResultsFile(filename *string) *os.File {
	fileResults, err := os.OpenFile(*filename, os.O_APPEND|os.O_RDWR, 0600)
	checkErr(err)
	return fileResults
}

func writeResults(hashArr map[string]string) {
	resultsFilename := getProgramName() + "_hash_results.csv"
	fmt.Println("[+] Writing results to CSV file: " + resultsFilename)
	var filepointer *os.File
	if _, err := os.Stat(resultsFilename); os.IsNotExist(err) {
		filepointer = makeResultsFile(&resultsFilename)
	} else {
		filepointer = updateResultsFile(&resultsFilename)
	}

	for hash, filename := range hashArr {
		fmt.Fprintf(filepointer, "md5hash,\"%s\",%s,%x,%s\r\n", filepath.Clean(filename),
			getProgramSeverity(), hash, getProgramName())
	}
	filepointer.Close()
}

func getFileHash(filename string) {
	fileHash, err := os.Open(filename)
	checkErr(err)
	defer fileHash.Close()

	hashObj := md5.New()
	if _, err := io.Copy(hashObj, fileHash); err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("md5sum: %x\n", hashObj.Sum(nil))
	HashArray[string(hashObj.Sum(nil))] = filename
}

func getStringHash(stringHash string) {
	hashObj := md5.New()
	io.WriteString(hashObj, stringHash)
	fmt.Println("%x", hashObj.Sum(nil))
	HashArray[string(hashObj.Sum(nil))] = stringHash
}

func setProgramName() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter name of program: ")
	ProgramName, _ = reader.ReadString('\r')
	fmt.Println("[+] Program name set to: " + getProgramName())
}

func setProgramSeverity() {
ResetSeverity:
	reader := bufio.NewReader(os.Stdin)
	// get program severity and check it's value matches those in a list
	fmt.Println("Enter the severity (INFORMATIONAL / LOW / MEDIUM / HIGH): ")
	ProgramSeverity, _ = reader.ReadString('\r')
	severityMap := map[string]bool{
		"INFORMATIONAL": true,
		"LOW":           true,
		"MEDIUM":        true,
		"HIGH":          true,
	}
	if severityMap[getProgramSeverity()] {
		fmt.Println("[+] Program severity set to: " + getProgramSeverity())
	} else {
		fmt.Printf("[-] Program severity '%s' is not valid.\n", getProgramSeverity())
		goto ResetSeverity
	}

}

func setProgramAttributes() {
	setProgramName()
	setProgramSeverity()
}

func getProgramName() string {
	return strings.ToTitle(strings.TrimSuffix(ProgramName, "\r"))
}

func getProgramSeverity() string {
	return strings.ToUpper(strings.TrimSuffix(ProgramSeverity, "\r"))
}

func getDirs(input_dir string, c chan int) {
	runtime.Gosched()
	ignoreFiles := map[string]bool{
		"hashFiles.exe": true,
		".gitignore":    true,
		"README":        true,
		"LICENSE":       true,
	}
	ignoreDirs := map[string]bool{
		".git": true,
	}
	fmt.Println("[ ] Generating hashes for dir: " + input_dir)
	contents, err := ioutil.ReadDir(input_dir)
	checkErr(err)

	for _, f := range contents {
		// fmt.Println(f.Name())
		_filepath := input_dir + "\\" + f.Name()
		item, err := os.Lstat(_filepath)
		checkErr(err)
		if item.Mode().IsRegular() {
			if !ignoreFiles[f.Name()] {
				getFileHash(_filepath)
			}
		} else if !ignoreDirs[f.Name()] {
			fmt.Println("[-] Need to scan dir: " + _filepath)
			// make a goroutine for subdirectories
			c <- 1
			go getDirs(_filepath, c)
		}
	}
	c <- 1
}

func main() {
	setProgramAttributes()
	c := make(chan int)
	go getDirs(".", c)
	x := <-c
	if x > 0 {
		writeResults(HashArray)
		fmt.Scanln()
	}
}
