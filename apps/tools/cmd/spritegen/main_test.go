package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSpriteRoundTripPreservesOpaqueBlackAndTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{A: 0})
	img.SetNRGBA(1, 0, color.NRGBA{A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 255, G: 100, A: 255})
	source, err := generate(img)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "sprite.png")
	if err := extractPNG(path, []byte(source)); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	restored, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	for x := 0; x < 3; x++ {
		want := color.NRGBAModel.Convert(img.At(x, 0))
		got := color.NRGBAModel.Convert(restored.At(x, 0))
		if got != want {
			t.Fatalf("pixel %d: got %v, want %v", x, got, want)
		}
	}
}
