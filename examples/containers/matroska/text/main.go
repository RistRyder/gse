package main

import (
	"fmt"

	"github.com/ristryder/gse/containers/matroska"
)

func main() {
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
	subtitleTrack := subtitleTracks[4]

	readPlainTextSubtitle(matroskaFile, subtitleTrack)
}

func progressCallback(position int64, total int64) {
	fmt.Printf("Position: %v / %v\n", position, total)
}

func readPlainTextSubtitle(matroskaFile *matroska.MatroskaFile, subtitleTrack matroska.MatroskaTrackInfo) {
	//Progress callback is optional
	subtitles, subtitlesErr := matroskaFile.Subtitle(uint64(subtitleTrack.TrackNumber), progressCallback)
	if subtitlesErr != nil {
		fmt.Println("Error retrieving subtitle: ", subtitlesErr)

		return
	}

	for i, line := range subtitles {
		text, textErr := line.Text(subtitleTrack)

		if textErr != nil {
			fmt.Println("Error reading subtitle line: ", textErr)
		} else {
			fmt.Printf("[%v][%v - %v] --> %v\n", i, line.Start, line.End(), text)
		}
	}
}
