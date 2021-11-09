package logger

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)

var Verbose bool = false

// Creates a standard log (use it for nonharmful and useful informations)
func Log(format string, a ...interface{}) {
	log.Printf(format, a...)
}

// Sends a warn (use it for pottential problem)
func Warn(format string, a ...interface{}) {
	color.Set(color.FgYellow)
	log.Printf("[WARN]: "+format, a...)
	color.Unset()
}

// Sends a warn (use it to inform about a real problem with a system but with no need to stop the service)
func Err(format string, a ...interface{}) {
	color.Set(color.FgHiRed)
	log.Printf("[ERR]: "+format, a...)
	color.Unset()
}

// We are fucked
func Fatal(format string, a ...interface{}) {
	color.Set(color.FgRed)
	log.Fatalf("[FATAL]: "+format, a...)
	color.Unset()
}

// Logs if verbose flag is true
func LogV(format string, a ...interface{}) {
	if Verbose {
		Log(format, a...)
	}
}

type Dots struct {
	iterationsPerDot int
	currIterations   int
}

func CreateDots(iterationsPerDot int) *Dots {
	return &Dots{iterationsPerDot, 0}
}

func (d *Dots) PrintDots() {
	d.currIterations++
	if d.currIterations >= d.iterationsPerDot {
		fmt.Print(".")
		d.currIterations = 0
	}
}
