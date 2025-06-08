package main

import (
	"fmt"
	"image/png"
	"os"
	"strconv"

	"github.com/ristryder/gse/bluraysup"
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

	readBluRaySupSubtitle(matroskaFile, subtitleTrack)
}

func readBluRaySupSubtitle(matroskaFile *matroska.MatroskaFile, subtitleTrack matroska.MatroskaTrackInfo) {
	pcsDatas, pcsDatasErr := bluraysup.ParseBluRaySupFromMatroska(subtitleTrack, *matroskaFile)
	if pcsDatasErr != nil {
		fmt.Println("Error reading BluRaySup: ", pcsDatasErr)
	}

	for i, pcsData := range pcsDatas {
		fmt.Printf("[%v][%v - %v] --> %v\n", i, pcsData.StartTime, pcsData.EndTime, pcsData.PcsObjects)

		bitmap := pcsData.GetBitmap()
		newPngFile, newPngFileErr := os.Create(strconv.Itoa(i) + ".png")
		if newPngFileErr != nil {
			fmt.Println("Error creating destination PNG file: ", newPngFileErr)
			return
		}

		defer newPngFile.Close()

		pngErr := png.Encode(newPngFile, bitmap)
		if pngErr != nil {
			fmt.Println("Error encoding PNG image: ", pngErr)

			return
		}

		//Arbitrarily stop after a few images
		if i >= 20 {
			break
		}
	}
}
