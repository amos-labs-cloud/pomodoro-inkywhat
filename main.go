package main

import (
	"encoding/csv"
	"fmt"
	"github.com/icodealot/noaa"
	"gopkg.in/gographics/imagick.v2/imagick"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/inky"
	"periph.io/x/host/v3"
	"strconv"
	"time"
)

// Location represents geographical and demographic information for a zip code
type Location struct {
	Zip            string
	Lat            float64
	Lng            float64
	City           string
	StateID        string
	StateName      string
	Zcta           string
	ParentZcta     string
	Population     int
	Density        float64
	CountyFips     string
	CountyName     string
	CountyWeights  string
	CountyNamesAll string
	CountyFipsAll  string
	Imprecise      bool
	Military       bool
	Timezone       string
}

// Panel represents an image panel with text
type panel struct {
	filename string
	text     string
	color    string
	width    int
	height   int
	fontSize float64
}

// calculateFontSize returns a font size that will take up approximately 90% of the panel dimensions
func calculateFontSize(width, height int, textLength int) float64 {
	// Estimate font size based on panel dimensions
	// Font sizes are typically measured in points where the height is about 1.5x the width
	// We'll use the smaller dimension to ensure text fits

	minDimension := width
	if height < width {
		minDimension = height
	}

	// Calculate font size to take up 90% of panel's smaller dimension
	// Divide by text length to account for horizontal space needed
	// The divisor 2.0 is an approximation factor for text aspect ratio
	fontSizeEstimate := float64(minDimension) / float64(max(1, textLength/2))

	return fontSizeEstimate
}

func createPanel(p panel) error {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	// Set the background color to white
	pw.SetColor("white")
	err := mw.NewImage(uint(p.width), uint(p.height), pw)
	if err != nil {
		return err
	}

	// Create a drawing wand
	dw := imagick.NewDrawingWand()
	defer dw.Destroy()

	// Set the text color
	pw.SetColor(p.color)
	dw.SetFillColor(pw)

	// Set the font size
	dw.SetFontSize(p.fontSize)

	// Calculate position to center text
	xPos := float64(p.width) / 2
	yPos := float64(p.height)/2 + p.fontSize/3 // Adjust vertically to account for baseline

	// Set text alignment to center
	dw.SetTextAlignment(imagick.ALIGN_CENTER)

	// Add the text to the image, centered
	dw.Annotation(xPos, yPos, p.text)

	// Draw the text on the image
	err = mw.DrawImage(dw)
	if err != nil {
		return err
	}

	// Set the image format to PNG
	err = mw.SetImageFormat("png")
	if err != nil {
		return err
	}

	// Write the image to a file
	err = mw.WriteImage(p.filename)
	if err != nil {
		return err
	}

	return nil
}

// createWeatherPanel generates a panel showing temperature in Fahrenheit with a degree symbol
func createWeatherPanel(temperature string, width, height int, filename string) error {
	// Format temperature with degree symbol (°)
	tempText := fmt.Sprintf("%s°F", temperature)

	weatherPanel := panel{
		filename: filename,
		text:     tempText,
		color:    "red",
		width:    width,
		height:   height,
		fontSize: calculateFontSize(width, height, len(tempText)),
	}

	return createPanel(weatherPanel)
}

// findLocationByZip reads uszips.csv and returns location data for the specified zip code
func findLocationByZip(zipCode string) (Location, error) {
	file, err := os.Open("uszips.csv")
	if err != nil {
		return Location{}, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row first to skip it
	_, err = reader.Read()
	if err != nil {
		return Location{}, fmt.Errorf("failed to read header: %v", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Location{}, fmt.Errorf("error reading record: %v", err)
		}

		// Check if this record matches our zip code
		if record[0] == zipCode {
			loc := Location{
				Zip:            record[0],
				City:           record[3],
				StateID:        record[4],
				StateName:      record[5],
				Zcta:           record[6],
				ParentZcta:     record[7],
				CountyFips:     record[10],
				CountyName:     record[11],
				CountyWeights:  record[12],
				CountyNamesAll: record[13],
				CountyFipsAll:  record[14],
				Timezone:       record[17],
			}

			// Parse numeric fields with error handling
			if lat, err := strconv.ParseFloat(record[1], 64); err == nil {
				loc.Lat = lat
			}
			if lng, err := strconv.ParseFloat(record[2], 64); err == nil {
				loc.Lng = lng
			}
			if pop, err := strconv.Atoi(record[8]); err == nil {
				loc.Population = pop
			}
			if density, err := strconv.ParseFloat(record[9], 64); err == nil {
				loc.Density = density
			}
			if record[15] == "true" {
				loc.Imprecise = true
			}
			if record[16] == "true" {
				loc.Military = true
			}

			return loc, nil
		}
	}

	return Location{}, fmt.Errorf("zip code %s not found", zipCode)
}

// createMontage combines the individual panel images into a single montage image
func createMontage(weatherFile, dateFile, timeFile, montageFile string) error {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	// Read the three individual panel images
	weatherWand := imagick.NewMagickWand()
	defer weatherWand.Destroy()
	err := weatherWand.ReadImage(weatherFile)
	if err != nil {
		return fmt.Errorf("failed to read weather image: %v", err)
	}

	dateWand := imagick.NewMagickWand()
	defer dateWand.Destroy()
	err = dateWand.ReadImage(dateFile)
	if err != nil {
		return fmt.Errorf("failed to read date image: %v", err)
	}

	timeWand := imagick.NewMagickWand()
	defer timeWand.Destroy()
	err = timeWand.ReadImage(timeFile)
	if err != nil {
		return fmt.Errorf("failed to read time image: %v", err)
	}

	// Create a new image with white background
	pw := imagick.NewPixelWand()
	defer pw.Destroy()
	pw.SetColor("white")
	err = mw.NewImage(400, 300, pw)
	if err != nil {
		return fmt.Errorf("failed to create montage canvas: %v", err)
	}

	// Composite weather panel in the top-left (0,0)
	err = mw.CompositeImage(weatherWand, imagick.COMPOSITE_OP_OVER, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to composite weather image: %v", err)
	}

	// Composite date panel in the top-right (200,0)
	err = mw.CompositeImage(dateWand, imagick.COMPOSITE_OP_OVER, 200, 0)
	if err != nil {
		return fmt.Errorf("failed to composite date image: %v", err)
	}

	// Composite time panel in the bottom (0,100) covering the bottom 2/3
	err = mw.CompositeImage(timeWand, imagick.COMPOSITE_OP_OVER, 0, 100)
	if err != nil {
		return fmt.Errorf("failed to composite time image: %v", err)
	}

	// Write the final montage to the output file
	err = mw.WriteImage(montageFile)
	if err != nil {
		return fmt.Errorf("failed to write montage image: %v", err)
	}

	return nil
}

// formatCurrentDate returns the current date in "DDD MM/DD" format (e.g., "Mon 01/15")
func formatCurrentDate() string {
	now := time.Now()
	dayAbbrev := now.Format("Mon")
	dateFormat := now.Format("01/02") // MM/DD format
	return fmt.Sprintf("%s %s", dayAbbrev, dateFormat)
}

// formatCurrentTime returns the current time in "HH:MM" format
func formatCurrentTime() string {
	now := time.Now()
	return now.Format("15:04") // 24-hour format
}

func drawDisplay(path string) error {
	spiPort := "SPI0.0"
	dcPin := "22"
	resetPin := "27"
	busyPin := "17"

	// Open and decode the image.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	/* #nosec G307 */
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return err
	}

	if _, err = host.Init(); err != nil {
		return err
	}

	log.Printf("Opening %s...", spiPort)
	b, err := spireg.Open(spiPort)
	if err != nil {
		return err
	}

	log.Printf("Opening pins...")
	dc := gpioreg.ByName(dcPin)
	if dc == nil {
		return fmt.Errorf("invalid DC pin name: %s", dcPin)
	}

	reset := gpioreg.ByName(resetPin)
	if reset == nil {
		return fmt.Errorf("invalid Reset pin name: %s", resetPin)
	}

	busy := gpioreg.ByName(busyPin)
	if busy == nil {
		return fmt.Errorf("invalid Busy pin name: %s", busyPin)
	}

	log.Printf("Creating inky...")
	dev, err := inky.New(b, dc, reset, busy, &inky.Opts{
		Model:       inky.WHAT,
		ModelColor:  inky.Yellow,
		BorderColor: inky.Black,
	})
	if err != nil {
		return err
	}

	log.Printf("Drawing image...")
	return dev.Draw(img.Bounds(), img, image.Point{})
}

func main() {
	imagick.Initialize()
	defer imagick.Terminate()

	// Find location data for zip code 55343
	location, err := findLocationByZip("55343")
	if err != nil {
		fmt.Printf("Error finding location: %v\n", err)
		return
	}

	// Display the location found
	fmt.Printf("Found location: %s, %s, %s (Lat: %f, Lng: %f)\n",
		location.City, location.StateID, location.Zip, location.Lat, location.Lng)

	forecast, err := noaa.HourlyForecast(
		strconv.FormatFloat(location.Lat, 'f', -1, 64),
		strconv.FormatFloat(location.Lng, 'f', -1, 64),
	)
	if err != nil {
		fmt.Printf("Error getting forecast: %v\n", err)
		return
	}

	// Define filenames for panel images
	weatherFile := "weather.png"
	dateFile := "date.png"
	timeFile := "time.png"
	montageFile := "display.png"

	// Create weather panel with temperature from forecast - 200x100 for top-left section
	err = createWeatherPanel(strconv.FormatFloat(forecast.Periods[0].Temperature, 'f', 0, 64), 200, 100, weatherFile)
	if err != nil {
		panic(err)
	}

	// Get formatted date string - 200x100 for top-right section
	formattedDate := formatCurrentDate()
	datePanel := panel{
		filename: dateFile,
		text:     formattedDate,
		color:    "black",
		width:    200,
		height:   100,
		fontSize: calculateFontSize(200, 100, len(formattedDate)),
	}
	err = createPanel(datePanel)
	if err != nil {
		panic(err)
	}

	// Get current time in HH:MM format - 400x200 for bottom section (2/3 of height)
	currentTime := formatCurrentTime()
	timePanel := panel{
		filename: timeFile,
		text:     currentTime,
		color:    "blue",
		width:    400,
		height:   200,
		fontSize: calculateFontSize(400, 200, len(currentTime)),
	}
	err = createPanel(timePanel)
	if err != nil {
		panic(err)
	}

	// Create the montage from the individual panel images
	err = createMontage(weatherFile, dateFile, timeFile, montageFile)
	if err != nil {
		fmt.Printf("Error creating montage: %v\n", err)
		return
	}

	fmt.Println("Montage created successfully: " + montageFile)

	err = drawDisplay(montageFile)
	if err != nil {
		panic(err)
	}

}
