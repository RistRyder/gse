# gse - Go Subtitle Edit

Go port of the .NET `libse` library used by Subtitle Edit: <https://github.com/SubtitleEdit/subtitleedit/tree/main/src/libse>

This library is pre-release under active development and attempts to maintain the same API as `libse`.

Currently the track information of an MKV file is available and individual subtitle tracks can be read.

## Examples
### Container Formats
| Container Format  | Example Location |
| ------------- | ------------- |
| Matroska | [examples/containers/matroska/main.go](https://github.com/RistRyder/gse/blob/main/examples/containers/matroska/main.go) |

## License
`gse` is licensed under the GNU LESSER GENERAL PUBLIC LICENSE Version 3, 
so it free to use for commercial software, as long as you don't modify the library itself. 
LGPL 3.0 allows linking to the library in a way that doesn't require you to open source your own code. 
This means that if you use `gse` in your project, you can keep your own code private, 
as long as you don't modify `gse` itself.