package subtitles

import (
	"strconv"
	"strings"

	"github.com/ristryder/gse/common"
)

type expectingLine int32

const (
	ExpectingLineNumber expectingLine = iota
	ExpectingTimeCodes  expectingLine = iota + 1
	ExpectingText       expectingLine = iota + 2
)

const (
	defaultSeparator string = " --> "
	whitespaceCutset string = "\n\t "
)

type SubRip struct {
	errors     []string
	isMsFrames bool
}

func (s *SubRip) tryReadTimeCodesLine(input string, paragraph common.Paragraph, validate bool) bool {
	str := strings.TrimLeft(input, "- ")
	if len(str) < 10 {
		return false
	}
	if _, parseErr := strconv.ParseUint(string(str[0]), 10, 64); parseErr != nil {
		return false
	}

	//Fix some badly formatted separator sequences - anything can happen if you manually edit ;)
	line := strings.ReplaceAll(input, "،", ",")
	line = strings.ReplaceAll(line, "", ",")
	line = strings.ReplaceAll(line, "¡", ",")
	line = strings.ReplaceAll(line, "\u200B", " ") //zero width space
	line = strings.ReplaceAll(line, "\uFEFF", " ") //zero width no-break space
	line = strings.ReplaceAll(line, " -> ", defaultSeparator)
	line = strings.ReplaceAll(line, " —> ", defaultSeparator)  //em-dash
	line = strings.ReplaceAll(line, " ——> ", defaultSeparator) //em-dash
	line = strings.ReplaceAll(line, " - > ", defaultSeparator)
	line = strings.ReplaceAll(line, " ->> ", defaultSeparator)
	line = strings.ReplaceAll(line, " -- > ", defaultSeparator)
	line = strings.ReplaceAll(line, " - -> ", defaultSeparator)
	line = strings.ReplaceAll(line, " -->> ", defaultSeparator)
	line = strings.ReplaceAll(line, " ---> ", defaultSeparator)
	line = strings.ReplaceAll(line, "  ", " ")
	line = strings.ReplaceAll(line, ": ", ":")
	line = strings.ReplaceAll(line, " :", ":")
	line = strings.TrimSpace(line)

	//Removed stuff after time codes - like subtitle position
	// - example of position info: 00:02:26,407 --> 00:02:31,356  X1:100 X2:100 Y1:100 Y2:100
	if len(line) > 30 {
		if string(line[29]) == " " {
			line = common.Substr(line, 0, 29)
		} else if string(line[28]) == " " {
			line = common.Substr(line, 0, 28)
		} else if string(line[27]) == " " {
			line = common.Substr(line, 0, 27)
		}
	}

	//Removes all extra spaces
	line = strings.ReplaceAll(line, " ", "")
	line = strings.ReplaceAll(line, "-->", defaultSeparator)
	line = strings.TrimSpace(line)
	if !strings.Contains(line, defaultSeparator) {
		line = strings.ReplaceAll(line, ">", defaultSeparator)
	}

	//Fix a few more cases of wrong time codes, seen this: 00.00.02,000 --> 00.00.04,000
	line = strings.ReplaceAll(line, ".", ":")
	if len(line) >= 29 && (string(line[8]) == ":" || string(line[8]) == ";") {
		line = common.Substr(line, 0, 8) + "," + common.SubstrAll(line, 9)
	}
	if len(line) >= 29 && len(line) <= 30 && (string(line[25]) == ":" || string(line[25]) == ";") {
		line = common.Substr(line, 0, 25) + "," + common.SubstrAll(line, 26)
	}

	//Allow missing hours as some (buggy) websites generate these time codes: 04:48,460 --> 04:52,364
	if len(line) == 23 && string(line[2]) == ":" && string(line[5]) == "," && string(line[9]) == " " && string(line[12]) == ">" && string(line[13]) == " " && string(line[16]) == ":" && string(line[19]) == "," {
		line = "00:" + common.Substr(line, 0, 14) + "00:" + common.SubstrAll(line, 14)
	}

	//TODO: Remainder
	return false
}

func (s *SubRip) Errors() string {
	return strings.Join(s.errors, "\n")
}

func (s *SubRip) Extension() string {
	return ".srt"
}

func (s *SubRip) IsMine(lines []string, fileName string) (bool, error) {
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "webvtt") {
		return false, nil
	}

	subtitle := &common.Subtitle{}
	loadErr := s.LoadSubtitle(subtitle, lines, fileName)
	if loadErr != nil {
		return false, loadErr
	}

	return len(subtitle.Paragraphs) > len(s.errors), nil
}

func (s *SubRip) LoadSubtitle(subtitle *common.Subtitle, lines []string, fileName string) error {
	//TODO: Do
	return nil
}

func (s *SubRip) Name() string {
	return "SubRip"
}

func (s *SubRip) ToText(subtitle *common.Subtitle, title string) string {
	return ""
}
