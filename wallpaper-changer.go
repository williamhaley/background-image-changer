package main

import (
	"golang.org/x/sys/windows/registry"

	"fmt"
	"log"
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

var ValidExtensions []string = []string{ "gif", "jpg", "bmp" }
var TmpFilePath string = os.TempDir() + "/wallpapers.tmp"

var (
	regex *regexp.Regexp
)

func main() {
	getWallpapers()

	setWallpaperStyle()

	path := `C:\Users\Family\Desktop\774.jpg`
	setWallpaper(path)
}

func WallpaperRegex() *regexp.Regexp {
	if regex != nil {
		return regex
	}

	regexExtensions := []string{}

	for _, extension := range ValidExtensions {
		log.Println("Extension:", extension)
		regexExtensions = append(
			regexExtensions,
			fmt.Sprintf(FileFormat, extension),
		)
	}

	regex = regexp.MustCompile(strings.Join(regexExtensions, "|"))

	return regex
}

func getWallpapers() {
	var tmpFile *os.File
	var err error

	tmpFile, err = os.Create(TmpFilePath)
	defer tmpFile.Close()
	if err != nil {
		log.Panic(err)
	}

	directories := []string{ `C:\Users\Family\Pictures` }
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
			if matched := WallpaperRegex().MatchString(path); matched {
				atime, mtime, ctime, err := statTimes(path)
				fmt.Println(atime, mtime, ctime)
				tmpFile.WriteString(path + "\n")
			}
			return nil
		})
	}
}

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

func setWallpaper(path string) {
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
