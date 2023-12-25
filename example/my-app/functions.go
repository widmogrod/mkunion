package main

import (
	"bytes"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	_ "github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/workflow"
	_ "github.com/widmogrod/mkunion/x/workflow"
	"golang.org/x/image/draw"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"os"
)

var functions = map[string]workflow.Function{
	"concat": func(body *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
		args := body.Args
		a, ok := schema.As[string](args[0])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		b, ok := schema.As[string](args[1])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		return &workflow.FunctionOutput{
			Result: schema.MkString(a + b),
		}, nil
	},
	"concat_error": func(body *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
		args := body.Args
		a, ok := schema.As[string](args[0])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		b, ok := schema.As[string](args[1])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		if rand.Float32() > 0.5 {
			return nil, fmt.Errorf("random error")
		}

		return &workflow.FunctionOutput{
			Result: schema.MkString(a + b),
		}, nil
	},
	"genimageb64": func(body *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
		args := body.Args
		_, ok := schema.As[string](args[0])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		f, err := os.OpenFile("public/img.png", os.O_RDONLY, 644)
		if err != nil {
			return nil, fmt.Errorf("opening file, %w", err)
		}
		defer f.Close()

		value, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("file read problem, %w", err)
		}

		return &workflow.FunctionOutput{
			Result: schema.MkBinary(value),
		}, nil
	},
	"resizeimgb64": func(body *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
		resize := func(img image.Image, newWidth, newHeight int) image.Image {
			//func scaleTo(src image.Image,
			//	rect image.Rectangle, scale draw.Scaler) image.Image {
			rect := image.Rect(0, 0, newWidth, newHeight)
			newImg := image.NewNRGBA(rect)
			draw.NearestNeighbor.Scale(newImg, rect, img, img.Bounds(), draw.Over, nil)
			return newImg
			//}
			//// Calculate the aspect ratio of the original image
			//aspectRatio := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
			//
			//// Calculate the new dimensions of the image while preserving aspect ratio
			//if aspectRatio > 1 {
			//	// The image is wider than it is tall
			//	newHeight = int(float64(newWidth) / aspectRatio)
			//} else {
			//	// The image is taller than it is wide
			//	newWidth = int(float64(newHeight) * aspectRatio)
			//}
			//
			//// Create a new image with the desired dimensions
			//newImg := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
			//
			//// Draw the original image onto the new image, scaling it to fit
			//draw.Draw(newImg, image.Rect(0, 0, newWidth, newHeight), img, image.Point{0, 0}, draw.Src)
			//
			//return newImg
		}

		args := body.Args

		// Extract base64 string from input arguments
		imgBytes, ok := schema.As[[]byte](args[0])
		if !ok {
			return nil, fmt.Errorf("expected base64Str:string, got %T", args[0])
		}

		// Extract newWidth and newHeight from input arguments
		newWidth, ok := schema.As[int](args[1])
		if !ok {
			return nil, fmt.Errorf("expected newWidth:int, got %T", args[1])
		}

		newHeight, ok := schema.As[int](args[2])
		if !ok {
			return nil, fmt.Errorf("expected newHeight:int, got %T", args[2])
		}

		// Decode the base64 string into an image
		//imgBytes, err := base64.StdEncoding.DecodeString(base64Str)
		//if err != nil {
		//	return nil, err
		//}

		img, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			return nil, err
		}

		// Resize the image
		resizedImg := resize(img, newWidth, newHeight)

		// Encode the resized image back into a base64 string
		buf := new(bytes.Buffer)

		// Determine the image format based on the original image type
		switch img.(type) {
		case *image.NRGBA, *image.RGBA:
			err = png.Encode(buf, resizedImg)
		case *image.YCbCr:
			err = jpeg.Encode(buf, resizedImg, nil)
		default:
			return nil, fmt.Errorf("unsupported image type: %T", img)
		}

		if err != nil {
			return nil, err
		}

		// Return the resized image as a base64 string
		return &workflow.FunctionOutput{
			Result: schema.MkBinary(buf.Bytes()),
		}, nil
	},
}
