package main

import (
	"golang.org/x/sys/windows/registry"
	"github.com/sirupsen/logrus"

	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var log = logrus.New()

const (
	SetDesktopWallpaper uint64 = 20
	UpdateIniFile uint64 = 0x01
	SendWinIniChange uint64 = 0x02
	FileFormat = `(?i:\Q.\E%v$)`
)

type Config struct {
	Directories []string `json:"directories"`
	Extensions []string `json:"extensions"`
	Log bool `json:"log"`
	Wait time.Duration `json:"wait"`
}

func main() {
	var tmpFilePath string = os.TempDir() + "/background-images.tmp"
	var config *Config = loadConfig(tmpFilePath)
	var regex *regexp.Regexp = imageRegex(config.Extensions)

	if config.Log {
		log.Println("Logging to file rather than stdout")
		file, err := os.OpenFile("background-image-changer.log", os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			log.Out = file
		} else {
			log.Info("Failed to log to file, using default stderr")
		}
	}

	buildImageList(config.Directories, tmpFilePath, regex)
	setWallpaperStyle()
	run(config.Wait * time.Second, tmpFilePath)
}

func run(wait time.Duration, tmpFilePath string) {
	var path string
	var tmpFile *os.File
	var err error
	var numImages int
	var lineNumber int

	tmpFile, err = os.Open(tmpFilePath)
	if err != nil {
		log.Fatal(err)
	}

	numImages = lineCount(tmpFile)
	tmpFile.Close()

	// TODO WFH What's better, opening this file once and resetting the
	// scanner, or reading the file each and every iteration?
	for {
		tmpFile, err = os.Open(tmpFilePath)
		if err != nil {
			log.Println(err)
		} else {
			lineNumber = (rand.Intn(numImages) + 1)
			log.Println("Use image at line:", lineNumber)
			path, _, err = readLine(tmpFile, lineNumber)
			if err != nil {
				log.Println(err)
			} else {
				setBackgroundImage(path)
			}
		}
		tmpFile.Close()
		time.Sleep(wait)
	}
}

func loadConfig(tmpFilePath string) *Config {
	var bytes []byte
	var err error

	if bytes, err = ioutil.ReadFile("background-image-changer.config.json"); err != nil {
		log.Fatal(err)
	}

	var config Config
	if err = json.Unmarshal(bytes, &config); err != nil {
		log.Fatal(err)
	}

	if config.Wait <= 0 {
		log.Println("Invalid 'wait' specified. Using 60")
		config.Wait = 60
	}

	return &config
}

func imageRegex(extensions []string) *regexp.Regexp {
	var regexExtensions []string

	for _, extension := range extensions {
		regexExtensions = append(
			regexExtensions,
			fmt.Sprintf(FileFormat, extension),
		)
	}

	return regexp.MustCompile(strings.Join(regexExtensions, "|"))
}

func buildImageList(directories []string, tmpFilePath string, regex *regexp.Regexp) {
	var tmpFile *os.File
	var err error

	err = os.Remove(tmpFilePath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	tmpFile, err = os.Create(tmpFilePath)
	defer tmpFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	for _, directory := range directories {
		log.Println("Scanning:", directory)

		// TODO WFH I want this to be as simple as possible
		// https://stackoverflow.com/questions/30693421/how-to-read-specific-line-of-file
		filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if matched := regex.MatchString(path); matched {
				log.Println("Path:", path)
				tmpFile.WriteString(path + "\r\n")
			}
			return nil
		})
	}

	tmpFile.Sync()
}

func lineCount(r io.Reader) int {
	var scanner *bufio.Scanner
	var counter int

	scanner = bufio.NewScanner(r)
	counter = 0

	for scanner.Scan() {
		counter++
	}

	return counter
}

// https://stackoverflow.com/questions/30693421/how-to-read-specific-line-of-file
// This is NOT 0 indexed. Starts at line 1
func readLine(r io.Reader, lineNum int) (string, int, error) {
	var lastLine int = 0
	var scanner *bufio.Scanner = bufio.NewScanner(r)

	for scanner.Scan() {
		lastLine++
		if lastLine == lineNum {
			return scanner.Text(), lastLine, scanner.Err()
		}
	}
	return "", lastLine, io.EOF
}

func setWallpaperStyle() {
	var err error
	var key registry.Key

	key, err = registry.OpenKey(registry.CURRENT_USER, `Control Panel\Desktop`, registry.SET_VALUE)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Close()

	err = key.SetStringValue("WallpaperStyle", "1")
	if err != nil {
		log.Fatal(err)
	}

	err = key.SetStringValue("TileWallpaper", "0")
	if err != nil {
		log.Fatal(err)
	}
}

func setBackgroundImage(path string) {
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Setting background:", path)

	libuser32, err := syscall.LoadLibrary("user32.dll")
	if err != nil {
		log.Fatal(err)
	}
	spi, err := syscall.GetProcAddress(libuser32, "SystemParametersInfoW")
	if err != nil {
		log.Fatal(err)
	}
	ret, _, err := syscall.Syscall6(spi, 4,
		uintptr(SetDesktopWallpaper),
		uintptr(0),
		uintptr(unsafe.Pointer(pathp)),
		uintptr(UpdateIniFile | SendWinIniChange),
		0, 0,
	)
	// err is always non-nil - check the return value instead.
	log.Println(ret, err)
	if ret == 0 {
		log.Fatal("Error calling SystemParametersInfo: " + err.Error())
	}
}

