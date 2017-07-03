package main

import (
	"golang.org/x/sys/windows/registry"

	"syscall"
	"unsafe"
	"log"
)

const (
	SetDesktopWallpaper uint64 = 20
	UpdateIniFile uint64 = 0x01
	SendWinIniChange uint64 = 0x02
)

func main() {
	path := `C:\Users\Family\Desktop\774.jpg`
	setWallpaperStyle()
	setWallpaper(path)
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
