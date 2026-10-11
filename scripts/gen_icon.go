package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	_ = os.MkdirAll("assets", 0755)
	generateAppIcon()
	generateTrayIcon()
}

func generateAppIcon() {
	const size = 512
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw rounded background with gradient
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r := 100.0
			dx := math.Min(float64(x), float64(size-1-x))
			dy := math.Min(float64(y), float64(size-1-y))

			inside := true
			if dx < r && dy < r {
				dist := math.Hypot(r-dx, r-dy)
				if dist > r {
					inside = false
				}
			}

			if inside {
				factor := float64(y) / float64(size)
				cr := uint8(26 + factor*15)
				cg := uint8(30 + factor*40)
				cb := uint8(46 + factor*80)
				img.Set(x, y, color.RGBA{R: cr, G: cg, B: cb, A: 255})
			} else {
				img.Set(x, y, color.Transparent)
			}
		}
	}

	centerX := 256.0
	centerY := 230.0
	outerR := 130.0
	innerR := 70.0

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x) - centerX
			fy := float64(y) - centerY
			dist := math.Hypot(fx, fy)

			inArc := fy >= 0 && dist >= innerR && dist <= outerR
			inLeftLeg := fy < 0 && fy >= -70 && fx >= -outerR && fx <= -innerR
			inRightLeg := fy < 0 && fy >= -70 && fx >= innerR && fx <= outerR

			inLeftTip := fy < -70 && fy >= -120 && fx >= -outerR && fx <= -innerR
			inRightTip := fy < -70 && fy >= -120 && fx >= innerR && fx <= outerR

			if inArc || inLeftLeg || inRightLeg {
				img.Set(x, y, color.RGBA{R: 233, G: 69, B: 96, A: 255})
			} else if inLeftTip || inRightTip {
				img.Set(x, y, color.RGBA{R: 224, G: 224, B: 224, A: 255})
			}
		}
	}

	for y := 200; y <= 400; y++ {
		for x := 200; x <= 312; x++ {
			fx := float64(x) - centerX
			fy := float64(y)

			if fy >= 210 && fy <= 340 && math.Abs(fx) <= 16 {
				img.Set(x, y, color.RGBA{R: 255, G: 215, B: 0, A: 255})
			}
			if fy > 340 && fy <= 400 {
				prog := (fy - 340) / 60.0
				allowedW := 55.0 * (1.0 - prog)
				if math.Abs(fx) <= allowedW {
					img.Set(x, y, color.RGBA{R: 255, G: 215, B: 0, A: 255})
				}
			}
		}
	}

	outPath := filepath.Join("assets", "icon.png")
	f, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	println("Created", outPath)
}

func generateTrayIcon() {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	centerX := 32.0
	centerY := 36.0
	outerR := 22.0
	innerR := 12.0

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x) - centerX
			fy := float64(y) - centerY
			dist := math.Hypot(fx, fy)

			// Horseshoe arc at bottom
			inArc := fy >= 0 && dist >= innerR && dist <= outerR
			// Legs going up
			inLeftLeg := fy < 0 && fy >= -20 && fx >= -outerR && fx <= -innerR
			inRightLeg := fy < 0 && fy >= -20 && fx >= innerR && fx <= outerR

			if inArc || inLeftLeg || inRightLeg {
				img.Set(x, y, white)
			} else {
				img.Set(x, y, color.Transparent)
			}
		}
	}

	outPath := filepath.Join("assets", "tray_icon.png")
	f, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	println("Created", outPath)
}
