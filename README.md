# gse - Go Subtitle Edit

Go port of the .NET `libse` library used by Subtitle Edit: <https://github.com/SubtitleEdit/subtitleedit/tree/main/src/libse>

This library is pre-release under active development and attempts to maintain the same API as `libse`.

## Example - Read Subtitle Track
Currently the track information of an MKV file is available and individual subtitle tracks can be read.
```go
func main() {
	matroskaFile, matroskaFileErr := NewMatroskaFile("/path/to/video/file.mkv")
	if matroskaFileErr != nil {
		fmt.Println("Error opening Matroska file: ", matroskaFileErr)

		return
	}

	defer matroskaFile.Close()

	if !matroskaFile.IsValid {
		fmt.Println("Matroska file is not valid.")

		return
	}

	subtitleTracks, subtitleTracksErr := matroskaFile.Tracks(true)
	if subtitleTracksErr != nil {
		fmt.Println("Error retrieving tracks: ", subtitleTracksErr)

		return
	}

	for i, track := range subtitleTracks {
		fmt.Printf("Track %d: %v\n", i, track)
	}

	//Arbitrarily select subtitle track
	subtitleTrackNumber := uint64(4)

	subtitles, subtitlesErr := matroskaFile.Subtitles(subtitleTrackNumber, progressCallback)
	if subtitlesErr != nil {
		fmt.Println("Error retrieving subtitle: ", subtitlesErr)

		return
	}

	subtitleTrack := subtitleTracks[subtitleTrackNumber]

	for i, line := range subtitles {
		text, textErr := line.Text(subtitleTrack)

		if textErr != nil {
			fmt.Println("Error reading subtitle line: ", textErr)
		} else {
			fmt.Printf("[%v][%v - %v] --> %v\n", i, line.Start, line.End(), text)
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