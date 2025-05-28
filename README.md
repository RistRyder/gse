# gse - Go Subtitle Edit

Go port of the .NET `libse` library used by Subtitle Edit: <https://github.com/SubtitleEdit/subtitleedit/tree/main/src/libse>

```go
func main() {
	file, err := NewMatroskaFile("/path/to/video/file.mkv")
	if err != nil {
		fmt.Println("Error opening file: ", err)

		return
	}

	defer file.Close()

	trackNumber := uint64(4)

	subtitles, subtitlesErr := file.Subtitles(trackNumber, progressCallback)
	if subtitlesErr != nil {
		fmt.Println("Subtitle Error: ", subtitlesErr)
	} else {
		for i, line := range subtitles {
			text, textErr := line.Text(file.Tracks[trackNumber])

			if textErr != nil {
				fmt.Println("Error: ", textErr)
			} else {
				fmt.Printf("[%v][%v - %v] --> %v\n", i, line.Start, line.End(), text)
			}
		}
	}
}

func progressCallback(position int64, total int64) {
	fmt.Printf("Position: %v / %v\n", position, total)
}
```

## License
`gse` is licensed under the GNU LESSER GENERAL PUBLIC LICENSE Version 3, 
so it free to use for commercial software, as long as you don't modify the library itself. 
LGPL 3.0 allows linking to the library in a way that doesn't require you to open source your own code. 
This means that if you use libse in your project, you can keep your own code private, 
as long as you don't modify libse itself.