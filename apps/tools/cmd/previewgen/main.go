package main

import (
	"flag"
	"fmt"
	"image"
	"image/gif"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	input := flag.String("input", "dist/clock-check", "runtime artifact directory")
	output := flag.String("output", "docs/previews", "gallery directory")
	flag.Parse()

	if err := run(*input, *output); err != nil {
		log.Fatal(err)
	}
}

func run(input, output string) error {
	for _, name := range []string{"cigar", "mackerel", "seagull"} {
		var latest string
		var modified time.Time
		err := filepath.WalkDir(input, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || entry.Name() != name+".gif" {
				return nil
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.ModTime().After(modified) {
				latest = path
				modified = info.ModTime()
			}

			return nil
		})
		if err != nil {
			return err
		}
		if latest == "" {
			return fmt.Errorf("no preview for %s; run make clock-check first", name)
		}

		if err := scaleGIF(latest, filepath.Join(output, name+".gif")); err != nil {
			return err
		}
	}

	return nil
}

func scaleGIF(source, target string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}

	animation, err := gif.DecodeAll(file)
	file.Close()
	if err != nil {
		return err
	}

	const scale = 6
	for index, frame := range animation.Image {
		bounds := frame.Bounds()
		if bounds.Dx() != 52 || bounds.Dy() != 16 {
			return fmt.Errorf("%s: expected a 52x16 frame", source)
		}

		enlarged := image.NewPaletted(image.Rect(0, 0, 52*scale, 16*scale), frame.Palette)
		for y := 0; y < enlarged.Rect.Dy(); y++ {
			for x := 0; x < enlarged.Rect.Dx(); x++ {
				enlarged.SetColorIndex(x, y, frame.ColorIndexAt(bounds.Min.X+x/scale, bounds.Min.Y+y/scale))
			}
		}

		animation.Image[index] = enlarged
	}

	animation.Config.Width *= scale
	animation.Config.Height *= scale

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	output, err := os.Create(target)
	if err != nil {
		return err
	}

	if err := gif.EncodeAll(output, animation); err != nil {
		output.Close()
		return err
	}

	return output.Close()
}
