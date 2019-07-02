package service

import (
	"github.com/gojek/darkroom/pkg/metrics"
	"github.com/gojek/darkroom/pkg/processor"
	"strconv"
	"time"
)

const (
	width        = "w"
	height       = "h"
	fit          = "fit"
	crop         = "crop"
	mono         = "mono"
	blackHexCode = "000000"

	cropDurationKey      = "cropDuration"
	resizeDurationKey    = "resizeDuration"
	grayScaleDurationKey = "grayScaleDuration"
)

// Manipulator interface sets the contract on the implementation for common processing support in darkroom
type Manipulator interface {
	// Process takes ProcessSpec as an argument and returns []byte, error
	Process(spec ProcessSpec) ([]byte, error)
}

type manipulator struct {
	processor processor.Processor
}

// ProcessSpec defines the specification for a image manipulation job
type ProcessSpec struct {
	// Scope defines a scope for the image manipulation job, it can be used for logging/mertrics collection purposes
	Scope     string
	// ImageData holds the actual image contents to processed
	ImageData []byte
	// Params hold the key-value pairs for the processing job and tells the manipulator what to do with the image
	Params    map[string]string
}

// Process takes ProcessSpec as an argument and returns []byte, error
// This manipulator uses bild to do the actual image manipulations
func (m *manipulator) Process(spec ProcessSpec) ([]byte, error) {
	params := spec.Params
	data := spec.ImageData
	var err error
	if params[fit] == crop {
		t := time.Now()
		data, err = m.processor.Crop(data, CleanInt(params[width]), CleanInt(params[height]), GetCropPoint(params[crop]))
		if err == nil {
			metrics.Update(metrics.UpdateOption{Name: cropDurationKey, Type: metrics.Duration, Duration: time.Since(t), Scope: spec.Scope})
		}
	} else if len(params[fit]) == 0 && (CleanInt(params[width]) != 0 || CleanInt(params[height]) != 0) {
		t := time.Now()
		data, err = m.processor.Resize(data, CleanInt(params[width]), CleanInt(params[height]))
		if err == nil {
			metrics.Update(metrics.UpdateOption{Name: resizeDurationKey, Type: metrics.Duration, Duration: time.Since(t), Scope: spec.Scope})
		}
	}
	if params[mono] == blackHexCode {
		t := time.Now()
		data, err = m.processor.GrayScale(data)
		if err == nil {
			metrics.Update(metrics.UpdateOption{Name: grayScaleDurationKey, Type: metrics.Duration, Duration: time.Since(t), Scope: spec.Scope})
		}
	}
	return data, err
}

// CleanInt takes a string and return an int not greater than 9999
func CleanInt(input string) int {
	val, _ := strconv.Atoi(input)
	if val <= 0 {
		return 0
	}
	return val % 10000 // Never return value greater than 9999
}

// GetCropPoint takes a string and returns the type CropPoint
func GetCropPoint(input string) processor.CropPoint {
	switch input {
	case "top":
		return processor.CropTop
	case "top,left":
		return processor.CropTopLeft
	case "top,right":
		return processor.CropTopRight
	case "left":
		return processor.CropLeft
	case "right":
		return processor.CropRight
	case "bottom":
		return processor.CropBottom
	case "bottom,left":
		return processor.CropBottomLeft
	case "bottom,right":
		return processor.CropBottomRight
	default:
		return processor.CropCenter
	}
}

// NewManipulator takes in a Processor interface and returns a new manipulator
func NewManipulator(processor processor.Processor) *manipulator {
	return &manipulator{processor: processor}
}
