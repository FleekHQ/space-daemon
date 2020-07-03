package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"
)

const (
	tag            = "v0.0.3"
	executableName = "space"
	binaryName     = "space-daemon"
	downloadURL    = "https://github.com/FleekHQ/space-daemon/releases/download/"
)

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	osName := strings.Title(runtime.GOOS)
	arch := "i386"

	if runtime.GOARCH == "amd64" {
		arch = "x86_64"
	}

	currExecutablePath, err := os.Executable()
	checkErr(err)

	pathSegments := strings.Split(currExecutablePath, "/")
	wd := strings.Join(pathSegments[:len(pathSegments)-1], "/")
	executablePath := wd + "/" + executableName

	fmt.Printf("Downloading Space Daemon in %s\n", executablePath)

	fileUrl := downloadURL + tag + "/" + binaryName + "_" + osName + "_" + arch
	err = DownloadFile(executablePath, fileUrl)
	checkErr(err)

	err = os.Chmod(executablePath, 0755)
	checkErr(err)

	// Uncomment if we need to use config JSON
	//err = config.CreateConfigJson()
	//checkErr(err)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nEnter your Textile User Key> ")
	textileUserKey, err := reader.ReadString('\n')
	checkErr(err)
	fmt.Print("\nEnter your Textile User Secret> ")
	textileUserSecret, err := reader.ReadString('\n')
	checkErr(err)

	fmt.Println("")
	fmt.Println("Creating .env file")
	envFile, err := os.Create(wd + "/" + ".env")
	checkErr(err)

	_, err = envFile.WriteString("TXL_USER_KEY=" + textileUserKey + "\nTXL_USER_SECRET=" + textileUserSecret + "\nLOG_LEVEL=DEBUG\n")
	checkErr(err)

	fmt.Println("Space Daemon installed successfully. Run ./space to start it.")
}

func DownloadFile(filepath string, url string) error {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}
