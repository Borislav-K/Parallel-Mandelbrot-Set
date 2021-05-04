package main

import (
	"flag"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/Borislav-K/Parallel-Mandelbrot-Set/rendering"
)

const (
	imageWidth    = 15000
	imageHeight   = 9000
	maxIterations = 256
)

var (
	pixels           [imageWidth][imageHeight]uint8
	taskDistribution = rand.Perm(imageWidth) // Only used for random partitioning (g=0)

	parallelism int
	granularity int

	startX float64
	startY float64
	endX   float64
	endY   float64

	threadTimeSpent []int64
	wg              sync.WaitGroup
)

func main() {
	parseConfig()
	log.Printf("Starting with parallelism %d and granularity %d\n", parallelism, granularity)

	threadTimeSpent = make([]int64, parallelism)
	wg.Add(parallelism)

	startTime := time.Now()
	for i := 1; i <= parallelism-1; i++ {
		startThread(i)
	}
	startThread(0)
	threadTimeSpent[0] = time.Now().Sub(startTime).Milliseconds()

	wg.Wait()
	log.Printf("Done calculating Mandelbrot set. Time elapsed: %d (millis)\n", time.Now().Sub(startTime).Milliseconds())
	log.Printf("Elapsed time for all threads: %+v.\n", threadTimeSpent)
	renderMandelbrotSet()
	log.Printf("Done rendering the picture. Time elapsed (total): %d (millis)\n", time.Now().Sub(startTime).Milliseconds())
	//rendering.ExportAsJPG(rendering.GraphThreadSegments(parallelism, granularity, imageWidth, imageHeight), "threads")
}

func startThread(i int) {
	if granularity == 0 {
		go calculateMandelbrotSetFragmentRandomDist(i)
	} else {
		go calculateMandelbrotSetFragment(i)
	}
}

func calculateMandelbrotSetFragment(i int) {
	// Each goroutine takes a rectangular segment of the whole image
	startTime := time.Now()

	taskSize := imageWidth / granularity / parallelism

	for taskN := 1; taskN <= granularity; taskN++ {
		startingRow := i*taskSize + (taskN-1)*imageWidth/granularity // small offset + big offset
		endingRow := startingRow + taskSize
		if i == (parallelism - 1) { // If it's the last thread
			// Finish the remainder of this section's rows (needed due to integer arithmetic)
			endingRow = taskN * imageWidth / granularity
		}
		for x := startingRow; x < endingRow; x++ {
			for y := 0; y < imageHeight; y++ {
				cx := startX + (endX-startX)*float64(x)/float64(imageWidth-1)
				cy := startY + (endY-startY)*float64(y)/float64(imageHeight-1)
				pixels[x][y] = calculateColour(cx, cy)
			}
		}
	}

	threadTimeSpent[i] = time.Now().Sub(startTime).Milliseconds()
	wg.Done()
}

func calculateMandelbrotSetFragmentRandomDist(i int) {
	startTime := time.Now()

	for rowIndex := i * (imageWidth / parallelism); rowIndex < (i+1)*(imageWidth/parallelism); rowIndex++ {
		x := taskDistribution[rowIndex]
		for y := 0; y < imageHeight; y++ {
			cx := startX + (endX-startX)*float64(x)/float64(imageWidth-1)
			cy := startY + (endY-startY)*float64(y)/float64(imageHeight-1)
			pixels[x][y] = calculateColour(cx, cy)
		}
	}

	threadTimeSpent[i] = time.Now().Sub(startTime).Milliseconds()
	wg.Done()
}

func calculateColour(cx, cy float64) uint8 {
	var x, y, xx, yy = 0.0, 0.0, 0.0, 0.0

	for i := 0; i < maxIterations; i++ {
		xy := x * y
		xx = x * x
		yy = y * y
		if xx+yy > 4 {
			return uint8(i)
		}
		x = xx - yy + cx
		y = 2*xy + cy
	}
	return 0
}

func renderMandelbrotSet() {
	img := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))
	for x := 0; x < imageWidth; x++ {
		for y := 0; y < imageHeight; y++ {
			img.Set(x, y, color.RGBA{
				R: 0,
				G: 0,
				B: pixels[x][y],
				A: math.MaxUint8,
			})
		}
	}

	rendering.ExportAsJPG(img, "result")
}

func parseConfig() {
	parallelism = runtime.GOMAXPROCS(0)
	flag.Float64Var(&startX, "startX", -0.87, "Starting X coordinates")
	flag.Float64Var(&startY, "startY", -0.215, "Starting Y coordinates")
	flag.Float64Var(&endX, "endX", -0.814, "Ending X coordinates")
	flag.Float64Var(&endY, "endY", -0.1976, "Ending Y coordinates")
	flag.IntVar(&granularity, "g", 1, "Granularity(>=0), if 0 then random partitioning is used")
	flag.Parse()
}
