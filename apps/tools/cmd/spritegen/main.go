package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const beginMarker = "    # sprite:begin"
const endMarker = "    # sprite:end"

func main() {
	appDir := flag.String("app-dir", "", "one screen directory (default: all PNG-backed screens)")
	check := flag.Bool("check", false, "fail when generated Berry differs")
	extract := flag.Bool("extract", false, "recover sprite.png from existing palette/rows")
	flag.Parse()

	if err := run(*appDir, *check, *extract); err != nil {
		log.Fatal(err)
	}
}

func run(appDir string, check, extract bool) error {
	var paths []string
	if appDir != "" {
		paths = []string{filepath.Join(appDir, "main.be")}
	} else {
		if extract {
			return fmt.Errorf("--extract requires --app-dir")
		}

		images, err := filepath.Glob("apps/clock/screens/*/sprite.png")
		if err != nil {
			return err
		}

		for _, path := range images {
			paths = append(paths, filepath.Join(filepath.Dir(path), "main.be"))
		}
	}

	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		pngPath := filepath.Join(filepath.Dir(path), "sprite.png")
		if extract {
			if err := extractPNG(pngPath, source); err != nil {
				return err
			}

			continue
		}

		data, err := os.ReadFile(pngPath)
		if err != nil {
			return err
		}

		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return err
		}

		generated, err := generate(img)
		if err != nil {
			return fmt.Errorf("%s: %w", pngPath, err)
		}

		start := bytes.Index(source, []byte(beginMarker))
		end := bytes.Index(source, []byte(endMarker))
		if start < 0 || end < start {
			return fmt.Errorf("%s: missing sprite markers", path)
		}

		updated := string(source[:start]) + generated + string(source[end+len(endMarker):])
		if check && updated != string(source) {
			return fmt.Errorf("%s: regenerate with make sprites", path)
		}

		if !check {
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func generate(img image.Image) (string, error) {
	bounds := img.Bounds()
	if bounds.Dx() < 1 || bounds.Dx() > 128 || bounds.Dy() < 1 || bounds.Dy() > 32 {
		return "", fmt.Errorf("sprite dimensions must fit 128x32")
	}

	palette := []uint32{0}
	indices := map[uint32]int{}
	rows := make([][]int, bounds.Dy())
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			rgba := color.NRGBAModel.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
			if rgba.A == 0 {
				rows[y] = append(rows[y], 0)
				continue
			}
			if rgba.A != 255 {
				return "", fmt.Errorf("partial transparency at %d,%d", x, y)
			}

			rgb := uint32(rgba.R)<<16 | uint32(rgba.G)<<8 | uint32(rgba.B)
			index, exists := indices[rgb]
			if !exists {
				if len(palette) == 256 {
					return "", fmt.Errorf("sprite exceeds 255 opaque colors")
				}
				index = len(palette)
				indices[rgb] = index
				palette = append(palette, rgb)
			}

			rows[y] = append(rows[y], index)
		}
	}

	var output strings.Builder
	fmt.Fprintln(&output, beginMarker)
	fmt.Fprintln(&output, "    # Generated from sprite.png by make sprites; edit the PNG instead.")
	fmt.Fprint(&output, "    self.colors = [")
	for index, rgb := range palette {
		if index > 0 {
			output.WriteString(", ")
		}
		fmt.Fprintf(&output, "0x%06X", rgb)
	}

	fmt.Fprintln(&output, "]\n    self.rows = [")
	for y, row := range rows {
		output.WriteString("      [")
		for x, index := range row {
			if x > 0 {
				output.WriteString(", ")
			}
			fmt.Fprint(&output, index)
		}
		output.WriteString("]")
		if y+1 < len(rows) {
			output.WriteString(",")
		}
		output.WriteByte('\n')
	}

	fmt.Fprintln(&output, "    ]")
	output.WriteString(endMarker)

	return output.String(), nil
}

func extractPNG(path string, source []byte) error {
	palettePattern := regexp.MustCompile(`self.colors = \[([^\]]+)\]`)
	rowsPattern := regexp.MustCompile(`(?s)self.rows = \[(.*?)\n    \]`)
	paletteMatch := palettePattern.FindSubmatch(source)
	rowsMatch := rowsPattern.FindSubmatch(source)
	if len(paletteMatch) != 2 || len(rowsMatch) != 2 {
		return fmt.Errorf("source has no palette/rows")
	}

	var palette []color.Color
	for index, raw := range strings.Split(string(paletteMatch[1]), ",") {
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 0, 32)
		if err != nil {
			return err
		}

		alpha := uint8(255)
		if index == 0 {
			alpha = 0
		}
		palette = append(palette, color.NRGBA{
			R: uint8(value >> 16),
			G: uint8(value >> 8),
			B: uint8(value),
			A: alpha,
		})
	}

	rowPattern := regexp.MustCompile(`\[([^\]]+)\]`)
	rows := rowPattern.FindAllSubmatch(rowsMatch[1], -1)
	if len(rows) == 0 {
		return fmt.Errorf("no sprite rows")
	}

	width := len(strings.Split(string(rows[0][1]), ","))
	img := image.NewPaletted(image.Rect(0, 0, width, len(rows)), palette)
	for y, row := range rows {
		values := strings.Split(string(row[1]), ",")
		if len(values) != width {
			return fmt.Errorf("inconsistent sprite row widths")
		}

		for x, raw := range values {
			index, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil || index < 0 || index >= len(palette) {
				return fmt.Errorf("invalid palette index at %d,%d", x, y)
			}

			img.SetColorIndex(x, y, uint8(index))
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}

	if err := png.Encode(file, img); err != nil {
		file.Close()
		return err
	}

	return file.Close()
}
