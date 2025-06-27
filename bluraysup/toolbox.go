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
	"fmt"
	"math"
)

// MillisecondsToTime converts time in milliseconds to an array with [hours, minutes, seconds, milliseconds]
func MillisecondsToTime(ms float64) [4]int64 {
	time := [4]int64{}

	//time[0] = hours
	time[0] = int64(math.Round(ms / (60 * 60 * 1000)))
	ms -= float64(time[0]) * 60 * 60 * 1000
	//time[1] = minutes
	time[1] = int64(math.Round(ms / (60 * 1000)))
	ms -= float64(time[1]) * 60 * 1000
	//time[2] = seconds
	time[2] = int64(math.Round(ms / (1000)))
	ms -= float64(time[2]) * 1000
	//time[3] = milliseconds
	time[3] = int64(math.Round(ms))

	return time
}

// PtsToTimeString converts time in 90kHz ticks to a string in "hh:mm:ss.ms" format
func PtsToTimeString(pts int64) string {
	time := MillisecondsToTime(float64(pts) / 90)

	return fmt.Sprintf("%2d:%2d:%2d.%3d", time[0], time[1], time[2], time[3])
}
