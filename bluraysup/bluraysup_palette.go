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

type BluRaySupPalette struct {
	size     int    //Number of palette entries
	r        []byte //Byte buffer for RED info
	g        []byte //Byte buffer for GREEN info
	b        []byte //Byte buffer for BLUE info
	a        []byte //Byte buffer for alpha info
	y        []byte //Byte buffer for Y (luminance) info
	cb       []byte //Byte buffer for Cb (chrominance blue) info
	cr       []byte //Byte buffer for Cr (chrominance red) info
	useBt601 bool   //Use BT.601 color model instead of BT.709
}

//Convert RGB color info to YCBCr
func rgb2YCbCr(r, g, b int, useBt601 bool) []int {
	yCbCr := make([]int, 3)
	var y, cb, cr float64

	if useBt601 {
		//BT.601 for RGB 0..255 (PC) -> YCbCr 16..235
		y = float64(r)*0.299*219/255 + float64(g)*0.587*219/255 + float64(b)*0.114*219/255
		cb = float64(-r)*0.168736*224/255 - float64(g)*0.331264*224/255 + float64(b)*0.5*224/255
		cr = float64(r)*0.5*224/255 - float64(g)*0.418688*224/255 - float64(b)*0.081312*224/255
	} else {
		//BT.709 for RGB 0..255 (PC) -> YCbCr 16..235
		y = float64(r)*0.2126*219/255 + float64(g)*0.7152*219/255 + float64(b)*0.0722*219/255
		cb = float64(-r)*0.2126/1.8556*224/255 - float64(g)*0.7152/1.8556*224/255 + float64(b)*0.5*224/255
		cr = float64(r)*0.5*224/255 - float64(g)*0.7152/1.5748*224/255 - float64(b)*0.0722/1.5748*224/255
	}

	yCbCr[0] = 16 + int(y+0.5)
	yCbCr[1] = 128 + int(cb+0.5)
	yCbCr[2] = 128 + int(cr+0.5)
	for i := 0; i < 3; i++ {
		if yCbCr[i] < 16 {
			yCbCr[i] = 16
		} else {
			if i == 0 {
				if yCbCr[i] > 235 {
					yCbCr[i] = 235
				}
			} else {
				if yCbCr[i] > 240 {
					yCbCr[i] = 240
				}
			}
		}
	}

	return yCbCr
}

//AlphaAtIndex returns the alpha channel at the specified palette index
func (b *BluRaySupPalette) AlphaAtIndex(index int) int {
	return int(b.a[index] & 0xFF)
}

//NewBluRaySupPalette initializes the palette with transparent black (RGBA: 0x00000000)
func NewBluRaySupPalette(palSize int, use601 bool) *BluRaySupPalette {
	palette := &BluRaySupPalette{
		size:     palSize,
		useBt601: use601,
		r:        make([]byte, palSize),
		g:        make([]byte, palSize),
		b:        make([]byte, palSize),
		a:        make([]byte, palSize),
		y:        make([]byte, palSize),
		cb:       make([]byte, palSize),
		cr:       make([]byte, palSize),
	}

	//set at least all alpha values to invisible
	yCbCr := rgb2YCbCr(0, 0, 0, use601)
	for i := 0; i < palSize; i++ {
		palette.y[i] = byte(yCbCr[0])
		palette.cb[i] = byte(yCbCr[1])
		palette.cr[i] = byte(yCbCr[2])
	}

	return palette
}

//NewDefaultBluRaySupPalette initializes the palette with transparent black (RGBA: 0x00000000)
func NewDefaultBluRaySupPalette(palSize int) *BluRaySupPalette {
	return NewBluRaySupPalette(palSize, false)
}

//SetAlpha sets the alpha channel at the specified palette index
func (b *BluRaySupPalette) SetAlpha(index, alpha int) {
	b.a[index] = byte(alpha)
}

//SetYCbCr sets the palette entry (YCbCr mode)
func (b *BluRaySupPalette) SetYCbCr(index, yn, cbn, crn int) {
	b.y[index] = byte(yn)
	b.cb[index] = byte(cbn)
	b.cr[index] = byte(crn)
	//create RGB
	rgb := YCbCr2Rgb(yn, cbn, crn, b.useBt601)
	b.r[index] = byte(rgb[0])
	b.g[index] = byte(rgb[1])
	b.b[index] = byte(rgb[2])
}

//YCbCr2Rgb converts YCbCr color info to RGB
func YCbCr2Rgb(y, cb, cr int, useBt601 bool) []int {
	rgb := make([]int, 3)
	var r, g, b float64

	y -= 16
	cb -= 128
	cr -= 128

	y1 := float64(y) * 1.164383562
	if useBt601 {
		//BT.601 for YCbCr 16..235 -> RGB 0..255 (PC)
		r = y1 + float64(cr)*1.596026317
		g = y1 - float64(cr)*0.8129674985 - float64(cb)*0.3917615979
		b = y1 + float64(cb)*2.017232218
	} else {
		//BT.709 for YCbCr 16..235 -> RGB 0..255 (PC)
		r = y1 + float64(cr)*1.792741071
		g = y1 - float64(cr)*0.5329093286 - float64(cb)*0.2132486143
		b = y1 + float64(cb)*2.112401786
	}

	rgb[0] = int(r + 0.5)
	rgb[1] = int(g + 0.5)
	rgb[2] = int(b + 0.5)
	for i := 0; i < 3; i++ {
		if rgb[i] < 0 {
			rgb[i] = 0
		} else if rgb[i] > 255 {
			rgb[i] = 255
		}
	}

	return rgb
}
