package main

import (
	"fmt"

	"github.com/RistRyder/gse/containers/matroska"
)

func main() {
	readPlainTextSubtitle()
}

func progressCallback(position int64, total int64) {
	fmt.Printf("Position: %v / %v\n", position, total)
}

func readPlainTextSubtitle() {
	matroskaFile, matroskaFileErr := matroska.NewMatroskaFile("/path/to/video/file.mkv")
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

	//Progress callback is optional
	subtitles, subtitlesErr := matroskaFile.Subtitle(subtitleTrackNumber, progressCallback)
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
