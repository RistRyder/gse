/*
 * Copyright 2009 Volker Oth (0xdeadbeef)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * NOTE: Converted to C# and modified by Nikse.dk@gmail.com
 * NOTE: Converted from C# to Go by github.com/RistRyder
 */

package bluraysup

import "image/draw"

const AlphaCrop = 14

func DecodeImage() *draw.Image {
	//TODO: Do
	return nil
}

func DecodePalette(paletteInfos []PaletteInfo) BluRaySupPalette {
	palette := NewDefaultBluRaySupPalette(256)
	//by definition, index 0xff is always completely transparent
	//also all entries must be fully transparent after initialization

	if len(paletteInfos) < 1 {
		return *palette
	}

	//always use last palette
	p := paletteInfos[len(paletteInfos)-1]

	fadeOut := false
	index := 0
	for i := 0; i < p.Size; i++ {
		//each palette entry consists of 5 bytes
		palIndex := p.Buffer[index]
		index++
		y := p.Buffer[index]
		index++
		cr := p.Buffer[index]
		index++
		cb := p.Buffer[index]
		index++
		alpha := p.Buffer[index]
		index++
		alphaOld := palette.AlphaAtIndex(int(palIndex))

		//avoid fading out
		if alpha >= byte(alphaOld) {
			if alpha < AlphaCrop {
				//to not mess with scaling algorithms, make transparent color black
				y = 16
				cr = 128
				cb = 128
			}

			palette.SetAlpha(int(palIndex), int(alpha))
		} else {
			fadeOut = true
		}

		palette.SetYCbCr(int(palIndex), int(y), int(cb), int(cr))
		index++
	}

	if fadeOut {
		//fade out detected -> patched palette\n
	}

	return *palette
}
