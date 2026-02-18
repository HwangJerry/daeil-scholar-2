// ImageResizeService — image processing (resize to max width)
package service

import "github.com/disintegration/imaging"

type ImageResizeService struct {
	maxWidth int
}

func NewImageResizeService(maxWidth int) *ImageResizeService {
	return &ImageResizeService{maxWidth: maxWidth}
}

// ImageDimensions holds width and height after processing.
type ImageDimensions struct {
	Width  int
	Height int
}

// ResizeIfNeeded opens the image at path, resizes if wider than maxWidth, and overwrites.
func (s *ImageResizeService) ResizeIfNeeded(path string) (*ImageDimensions, error) {
	img, err := imaging.Open(path)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w > s.maxWidth {
		img = imaging.Resize(img, s.maxWidth, 0, imaging.Lanczos)
		if err := imaging.Save(img, path); err != nil {
			return nil, err
		}
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
	}
	return &ImageDimensions{Width: w, Height: h}, nil
}
