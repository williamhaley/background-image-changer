# Windows Background Image Changer

I created this to cycle through family photos as the Windows desktop background (wallpaper).

This seemed like a feature Windows should provide out of the box, but I found its ability to handle subfolders lacking.

There is probably something in the Windows store that does this, but I took it as a chance to experiment with registry manipulation in Go.

# Config

See the `background-image-changer.config.json` file.

### Wait

The time duration, in seconds, to wait before cycling to the next image.

```
...
"wait": 10
...
```

### Directories

Paths to directories containing images. Directories will be recursively searched. Double backslashes are required.

```
...
"directories": [ "C:\\Photos", "E:\\Photos", "C:\\Users\\Will\\Pictures" ]
...
```

# Sources

* https://gist.github.com/christophberger/0eb5f12e3d6638c5f32d
* https://github.com/EmpireProject/Empire/blob/master/data/module_source/fun/Set-Wallpaper.ps1
* https://play.golang.org/p/liDl6OgI0y

