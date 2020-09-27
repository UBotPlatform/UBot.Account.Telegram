package main

import (
	"strings"
)

func guessImageExtByMIMEType(t string, dv string) string {
	loweredT := strings.ToLower(t)
	switch loweredT {
	case "image/pjpeg":
		fallthrough
	case "image/jpeg":
		return ".jpg"

	case "image/x-png":
		fallthrough
	case "image/png":
		return ".png"

	case "image/gif":
		return ".gif"

	case "image/bmp":
		return ".bmp"

	case "image/webp":
		return ".webp"

	case "image/tiff":
		return ".tif"

	default:
		return dv
	}
}

func guessImageExtByBytes(binary []byte, dv string) string {
	r := dv
	if len(binary) >= 4 {
		if binary[0] == 0xff && binary[1] == 0xd8 && binary[2] == 0xff {
			r = ".jpg"
		} else if binary[0] == 0x89 && binary[1] == 0x50 && binary[2] == 0x4e && binary[3] == 0x47 {
			r = ".png"
		} else if binary[0] == 0x47 && binary[1] == 0x49 && binary[2] == 0x46 && binary[3] == 0x38 {
			r = ".gif"
		} else if binary[0] == 0x42 && binary[1] == 0x4d {
			r = ".bmp"
		} else if binary[0] == 0x52 && binary[1] == 0x49 && binary[2] == 0x46 && binary[3] == 0x46 {
			//RIFF
			if len(binary) >= 12 {
				if binary[8] == 0x57 && binary[9] == 0x45 && binary[10] == 0x42 && binary[11] == 0x50 {
					r = ".webp"
				}
			}
		} else if binary[0] == 0x49 && binary[1] == 0x49 && binary[2] == 0x2a && binary[3] == 0x00 {
			r = ".tif"
		} else if binary[0] == 0x4d && binary[1] == 0x4d && binary[2] == 0x00 && binary[3] == 0x2a {
			r = ".tif"
		}
	}
	return r
}
