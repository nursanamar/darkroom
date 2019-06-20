package native

import (
	"bytes"
	"github.com/anthonynsimon/bild/clone"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/transform"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"***REMOVED***/darkroom/core/pkg/metrics"
	"***REMOVED***/darkroom/core/pkg/processor"
	"time"
)

const (
	pngType = "png"
	jpgType = "jpeg"

	cropDurationKey      = "cropDuration"
	resizeDurationKey    = "resizeDuration"
	watermarkDurationKey = "watermarkDuration"
	grayScaleDurationKey = "grayScaleDuration"
	decodeDurationKey    = "decodeDuration"
	encodeDurationKey    = "encodeDuration"
)

// BildProcessor uses bild library to process images using native Golang image.Image interface
type BildProcessor struct {
}

func (bp *BildProcessor) Crop(input []byte, width, height int, point processor.CropPoint) ([]byte, error) {
	img, f, err := bp.decode(input)
	if err != nil {
		return nil, err
	}

	w, h := getResizeWidthAndHeightForCrop(width, height, img.Bounds().Dx(), img.Bounds().Dy())

	t := time.Now()
	img = transform.Resize(img, w, h, transform.Linear)
	x0, y0 := getStartingPointForCrop(w, h, width, height, point)
	rect := image.Rect(x0, y0, width+x0, height+y0)
	img = (clone.AsRGBA(img)).SubImage(rect)
	metrics.Update(metrics.UpdateOption{Name: cropDurationKey, Type: metrics.Duration, Duration: time.Since(t)})

	return bp.encode(img, f)
}

func (bp *BildProcessor) Resize(input []byte, width, height int) ([]byte, error) {
	img, f, err := bp.decode(input)
	if err != nil {
		return nil, err
	}

	initW := img.Bounds().Dx()
	initH := img.Bounds().Dy()

	w, h := getResizeWidthAndHeight(width, height, initW, initH)
	if w != initW || h != initH {
		t := time.Now()
		img = transform.Resize(img, w, h, transform.Linear)
		metrics.Update(metrics.UpdateOption{Name: resizeDurationKey, Type: metrics.Duration, Duration: time.Since(t)})
	}

	return bp.encode(img, f)
}

func (bp *BildProcessor) Watermark(base []byte, overlay []byte, opacity uint8) ([]byte, error) {
	baseImg, f, err := bp.decode(base)
	if err != nil {
		return nil, err
	}
	overlayImg, _, err := bp.decode(overlay)
	if err != nil {
		return nil, err
	}

	t := time.Now()
	ratio := float64(overlayImg.Bounds().Dy()) / float64(overlayImg.Bounds().Dx())
	dWidth := float64(baseImg.Bounds().Dx()) / 2.0

	// Resizing overlay image according to base image
	overlayImg = transform.Resize(overlayImg, int(dWidth), int(dWidth*ratio), transform.Linear)

	// Anchor point for overlaying
	x := (baseImg.Bounds().Dx() - overlayImg.Bounds().Dx()) / 2
	y := (baseImg.Bounds().Dy() - overlayImg.Bounds().Dy()) / 2
	offset := image.Pt(int(x), int(y))

	// Mask image (that is just a solid light gray image)
	mask := image.NewUniform(color.Alpha{A: opacity})

	// Performing overlay
	draw.DrawMask(baseImg.(draw.Image), overlayImg.Bounds().Add(offset), overlayImg, image.ZP, mask, image.ZP, draw.Over)
	metrics.Update(metrics.UpdateOption{Name: watermarkDurationKey, Type: metrics.Duration, Duration: time.Since(t)})

	return bp.encode(baseImg, f)
}

func (bp *BildProcessor) GrayScale(input []byte) ([]byte, error) {
	img, f, err := bp.decode(input)
	if err != nil {
		return nil, err
	}

	t := time.Now()
	img = effect.Grayscale(img)
	metrics.Update(metrics.UpdateOption{Name: grayScaleDurationKey, Type: metrics.Duration, Duration: time.Since(t)})

	return bp.encode(img, f)
}

func (bp *BildProcessor) decode(data []byte) (image.Image, string, error) {
	t := time.Now()
	img, f, err := image.Decode(bytes.NewReader(data))
	if err == nil {
		metrics.Update(metrics.UpdateOption{Name: decodeDurationKey, Type: metrics.Duration, Duration: time.Since(t)})
	}
	return img, f, err
}

func (bp *BildProcessor) encode(img image.Image, format string) ([]byte, error) {
	t := time.Now()
	if format == pngType && isOpaque(img) {
		format = jpgType
	}
	buff := &bytes.Buffer{}
	var err error
	if format == pngType {
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(buff, img)
	} else {
		err = jpeg.Encode(buff, img, nil)
	}
	metrics.Update(metrics.UpdateOption{Name: encodeDurationKey, Type: metrics.Duration, Duration: time.Since(t)})
	return buff.Bytes(), err
}

func NewBildProcessor() *BildProcessor {
	return &BildProcessor{}
}
