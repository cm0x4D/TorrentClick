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
	const size = 512
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw rounded background with gradient
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Check corner radius
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
				// Gradient from deep navy (#1a1e2e) to vibrant electric blue (#0f3460)
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

	// Draw Magnet Horseshoe shape
	// Outer arc, inner arc, poles
	centerX := 256.0
	centerY := 230.0
	outerR := 130.0
	innerR := 70.0

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x) - centerX
			fy := float64(y) - centerY
			dist := math.Hypot(fx, fy)

			// Horseshoe bottom arc
			inArc := fy >= 0 && dist >= innerR && dist <= outerR
			// Horseshoe legs going up
			inLeftLeg := fy < 0 && fy >= -70 && fx >= -outerR && fx <= -innerR
			inRightLeg := fy < 0 && fy >= -70 && fx >= innerR && fx <= outerR

			// Silver tips / poles
			inLeftTip := fy < -70 && fy >= -120 && fx >= -outerR && fx <= -innerR
			inRightTip := fy < -70 && fy >= -120 && fx >= innerR && fx <= outerR

			if inArc || inLeftLeg || inRightLeg {
				// Red magnet body (#E94560)
				img.Set(x, y, color.RGBA{R: 233, G: 69, B: 96, A: 255})
			} else if inLeftTip || inRightTip {
				// Silver pole (#E0E0E0)
				img.Set(x, y, color.RGBA{R: 224, G: 224, B: 224, A: 255})
			}
		}
	}

	// Draw downward lightning / download arrow in center
	// Arrow points down from y=240 to y=380
	for y := 200; y <= 400; y++ {
		for x := 200; x <= 312; x++ {
			fx := float64(x) - centerX
			fy := float64(y)

			// Arrow shaft
			if fy >= 210 && fy <= 340 && math.Abs(fx) <= 16 {
				img.Set(x, y, color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold
			}
			// Arrow head
			if fy > 340 && fy <= 400 {
				prog := (fy - 340) / 60.0
				allowedW := 55.0 * (1.0 - prog)
				if math.Abs(fx) <= allowedW {
					img.Set(x, y, color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold
				}
			}
		}
	}

	_ = os.MkdirAll("assets", 0755)
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
