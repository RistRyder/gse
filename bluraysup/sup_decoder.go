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

import (
	"image"
	"image/color"
)

const AlphaCrop = 14

func putPixel(bmp *image.RGBA, index int, color color.RGBA) {
	if color.A > 0 {
		size := bmp.Rect.Size()
		x := index % size.X
		y := index / size.X
		if x < size.X && y < size.Y {
			bmp.Set(x, y, color)
		}
	}
}

func putPixelWithPalette(bmp *image.RGBA, index int, color int, palette BluRaySupPalette) {
	size := bmp.Rect.Size()
	x := index % size.X
	y := index / size.X
	if x < size.X && y < size.Y {
		bmp.Set(x, y, palette.ArgbColor(color))
	}
}

func DecodeImage(pcs PcsObject, data []OdsData, palettes []PaletteInfo) image.Image {
	if len(data) < 1 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	w := data[0].Size.Width
	h := data[0].Size.Height

	if w <= 0 || h <= 0 || len(data[0].Fragment.ImageBuffer) < 1 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	bm := image.NewRGBA(image.Rect(0, 0, w, h))
	pal := DecodePalette(palettes)

	ofs := 0
	xpos := 0
	index := 0

	buf := data[0].Fragment.ImageBuffer

	for {
		b := int(buf[index]) & 0xFF
		index++
		if b == 0 && index < len(buf) {
			b = int(buf[index]) & 0xFF
			index++
			if b == 0 {
				//next line
				ofs = ofs / w * w
				if xpos < w {
					ofs += w
				}

				xpos = 0
			} else {
				size := 0
				if (b & 0xC0) == 0x40 {
					if index < len(buf) {
						//00 4x xx -> xxx zeroes
						size = ((b - 0x40) << 8) + (int(buf[index]) & 0xFF)
						index++
						c := pal.ArgbColor(0)
						for i := 0; i < size; i++ {
							putPixel(bm, ofs, c)
							ofs++
						}

						xpos += size
					}
				} else if (b & 0xC0) == 0x80 {
					if index < len(buf) {
						//00 8x yy -> x times value y
						size = b - 0x80
						b = int(buf[index]) & 0xFF
						index++
						c := pal.ArgbColor(b)
						for i := 0; i < size; i++ {
							putPixel(bm, ofs, c)
							ofs++
						}

						xpos += size
					}
				} else if (b & 0xC0) != 0 {
					if index < len(buf) {
						//00 cx yy zz -> xyy times value z
						size = ((b - 0xC0) << 8) + (int(buf[index]) & 0xFF)
						index++
						b = int(buf[index]) & 0xFF
						index++
						c := pal.ArgbColor(b)
						for i := 0; i < size; i++ {
							putPixel(bm, ofs, c)
							ofs++
						}

						xpos += size
					}
				} else {
					//00 xx -> xx times 0
					c := pal.ArgbColor(0)
					for i := 0; i < b; i++ {
						putPixel(bm, ofs, c)
						ofs++
					}

					xpos += b
				}
			}
		} else {
			putPixelWithPalette(bm, ofs, b, pal)
			ofs++
			xpos++
		}

		if index >= len(buf) {
			break
		}
	}

	return bm
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
