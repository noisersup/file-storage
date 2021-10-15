package logger

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)

type Logger struct {
}

// Creates a standard log (use it for nonharmful and useful informations)
func (l *Logger) Log(format string, a ...interface{}) {
	log.Printf(format, a...)
}

// Sends a warn (use it for pottential problem)
func (l *Logger) Warn(format string, a ...interface{}) {
	color.Set(color.FgYellow)
	log.Printf("[WARN]: "+format, a...)
	color.Unset()
}

// Sends a warn (use it to inform about a real problem with a system but with no need to stop the service)
func (l *Logger) Err(format string, a ...interface{}) {
	color.Set(color.FgHiRed)
	log.Printf("[ERR]: "+format, a...)
	color.Unset()
}

// We are fucked
// (m is method or line number or file or path or whatever you want)
func (l *Logger) Fatal(format string, a ...interface{}) {
	color.Set(color.FgRed)
	log.Fatalf("[FATAL]: "+format, a...)
	color.Unset()
}

// Creates a standard log (use it for nonharmful and useful informations)
// (m is method or line number or file or path or whatever you want)
func (l *Logger) SLog(m, format string, a ...interface{}) {
	log.Printf(m+": "+format, a...)
}

// Sends a warn (use it for pottential problem)
// (m is method or line number or file or path or whatever you want)
func (l *Logger) SWarn(m, format string, a ...interface{}) {
	color.Set(color.FgYellow)
	log.Printf("[WARN] "+m+": "+format, a...)
	color.Unset()
}

// Sends a warn (use it to inform about a real problem with a system but with no need to stop the service)
// (m is method or line number or file or path or whatever you want)
func (l *Logger) SErr(m, format string, a ...interface{}) {
	color.Set(color.FgHiRed)
	log.Printf("[ERR] "+m+": "+format, a...)
	color.Unset()
}

// We are fucked
//(m is method or line number or file or path or whatever you want)
func (l *Logger) SFatal(m, format string, a ...interface{}) {
	color.Set(color.FgRed)
	log.Fatalf("[FATAL] "+m+": "+format, a...)
	color.Unset()
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
