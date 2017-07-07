package main

import (
	"golang.org/x/sys/windows/registry"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	SetDesktopWallpaper uint64 = 20
	UpdateIniFile uint64 = 0x01
	SendWinIniChange uint64 = 0x02
	FileFormat = `(?i:\Q.\E%v$)`
)

type Config struct {
	Directories []string `json:"directories"`
	Extensions []string `json:"extensions"`
	Wait time.Duration `json:"wait"`
}

func main() {
	var config *Config
	var regex *regexp.Regexp

	var tmpFilePath string = os.TempDir() + "/background-images.tmp"

	config = loadConfig(tmpFilePath)
	regex = imageRegex(config.Extensions)

	buildImageList(config.Directories, tmpFilePath, regex)
	setWallpaperStyle()
	run(config.Wait * time.Second, tmpFilePath)
}

func run(wait time.Duration, tmpFilePath string) {
	var path string

	for {
		path = getRandomImage(tmpFilePath)
		setBackgroundImage(path)
		time.Sleep(wait)
	}
}

func loadConfig(tmpFilePath string) *Config {
	var bytes []byte
	var err error

	if bytes, err = ioutil.ReadFile("background-image-changer.config.json"); err != nil {
		panic(err)
	}

	var config Config
	if err = json.Unmarshal(bytes, &config); err != nil {
		panic(err)
	}

	err = os.Remove(tmpFilePath)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
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

	tmpFile, err = os.Create(tmpFilePath)
	defer tmpFile.Close()
	if err != nil {
		log.Panic(err)
	}

	for _, directory := range directories {
		log.Println("Scanning:", directory)

		// TODO WFH What's the best way to store these files? I want to reduce
		// the number of times I hammer the disk to get files, so I want a tmp
		// file. But then if I have to re-scan it for changes... Do I blow
		// away the tmp file each scan? Do I Dynamically append to it? I also
		// want to be able to say something like "Show recent photos more often"
		// Which means I have to sort the order of the files in the tmp file.
		// I don't want this in memory. It's a wallpaper changer. It should be
		// tiny.
		// https://stackoverflow.com/questions/30693421/how-to-read-specific-line-of-file
		filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			//log.Println("Path:", path)
			if matched := regex.MatchString(path); matched {
				//atime, mtime, ctime, err := statTimes(path)
				//fmt.Println(atime, mtime, ctime)
				tmpFile.WriteString(path + "|")
			}
			return nil
		})
	}

	tmpFile.Sync()
}


/*
func statTimes(name string) (atime, mtime, ctime time.Time, err error) {
    fi, err := os.Stat(name)
    if err != nil {
        return
    }
    mtime = fi.ModTime()
    stat := fi.Sys().(*syscall.Stat_t)
    atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
    ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
    return
}
*/

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

	log.Println("Path:", path)

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
		panic("Error calling SystemParametersInfo: " + err.Error())
	}
}

func getRandomImage(tmpFilePath string) string {
	file, err := os.Open(tmpFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	offset := rand.Int63n(info.Size())
	foundPipe := false
	data := make([]byte, 1)
	path := ""

	for offset < info.Size() {
		_, err := file.ReadAt(data, offset)
		if err != nil {
			log.Fatal(err)
			continue
		}
		// log.Printf("read %d bytes: %q\n", count, data[0])

		char := byte(data[0])

		// Second time we're seeing a pipe, so we're done.
		if char == '|' && foundPipe {
			break
		}

		// We're in the midst of a hit. Append the data.
		if foundPipe {
			path += string(char)
		}

		// First time seeing a pipe. Start tracking chars.
		if char == '|' {
			foundPipe = true
		}

		offset++
		if offset >= info.Size()-1 {
			offset = 0
			path = ""
		}
	}

	return path
}

