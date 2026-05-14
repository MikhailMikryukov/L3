package services

import (
	"L3.4/internal/entities"
	"L3.4/pkg/messaging"
	"bytes"
	"context"
	"fmt"
	"github.com/michaelwp/goWatermark"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"strings"
)

// TasksRepository интерфейс обращения в бд
type TasksRepository interface {
	Create(ctx context.Context, task *entities.Task) error
	CheckStatus(ctx context.Context, id string) (string, error)
	UpdateStatus(ctx context.Context, taskMsg messaging.TaskMessage, status string, variants *entities.ImageVariants) error
	GetTaskByID(ctx context.Context, id string) (*entities.Task, error)
	Delete(ctx context.Context, task *entities.Task) error
}

// ImageProcessor обработчик изображений
type ImageProcessor struct {
	rep     TasksRepository
	storage *StorageService
}

// NewImageProcessor создание
func NewImageProcessor(rep TasksRepository, storage *StorageService) *ImageProcessor {
	return &ImageProcessor{
		rep:     rep,
		storage: storage,
	}
}

// SaveImage сохранить изображение
func (p *ImageProcessor) SaveImage(file multipart.File, header *multipart.FileHeader) error {

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	_, err = p.storage.SaveTemp(data, header.Filename)
	if err != nil {
		return err
	}

	return nil
}

// ProcessImage обработать изображение
func (p *ImageProcessor) ProcessImage(task *entities.Task) (*entities.ImageVariants, error) {
	// Загружаем оригинал
	imgData, err := p.storage.GetTemp(task.OriginalPath)
	if err != nil {
		return nil, err
	}

	imgSrc, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("error decode image %v", err)
	}
	var imgVariants entities.ImageVariants

	originalPath := fmt.Sprintf("originals/%s.jpg", task.ID)
	originalData, err := p.encodeImage(imgSrc, format, 100)
	if err != nil {
		return nil, fmt.Errorf("error encode image %v", err)
	}

	// Загружаем оригинальное изображение в хранилище
	ctx := context.Background()
	originalURL, err := p.storage.Upload(ctx, originalPath, originalData, "image/jpeg")

	imgVariants.Original = originalURL

	// Если указны параметры ресайза
	if len(task.Params.Width) > 0 {
		// Создаем миниатюру
		thumbnail := resize.Thumbnail(150, 150, imgSrc, resize.NearestNeighbor)
		thumbnailPath := fmt.Sprintf("thumbnails/%s.jpg", task.ID)
		thumbData, err := p.encodeImage(thumbnail, format, 100)
		if err != nil {
			return nil, fmt.Errorf("error encode image %v", err)
		}
		// Загружаем миниатюру в хранилище
		thumbnailURL, err := p.storage.Upload(ctx, thumbnailPath, thumbData, "image/jpeg")
		if err != nil {
			return nil, fmt.Errorf("error upload image %v", err)
		}

		imgVariants.Thumbnail = thumbnailURL

		// Делаем ресайз
		imgVariants.Resized = make(map[int]string)
		for _, w := range task.Params.Width {
			resized := resize.Resize(uint(w), uint(w), imgSrc, resize.Lanczos2)
			resizedPath := fmt.Sprintf("resized/%d_%s.jpg", w, task.ID)
			resizedData, err := p.encodeImage(resized, format, 100)
			if err != nil {
				return nil, fmt.Errorf("error encode image %v", err)
			}

			resizedURL, err := p.storage.Upload(ctx, resizedPath, resizedData, "image/jpeg")
			if err != nil {
				return nil, fmt.Errorf("error upload image %v", err)
			}
			// Загружаем ресайз в хранилище
			imgVariants.Resized[w] = resizedURL
		}

	}

	// Добавляем водяной знак
	if task.Params.AddWatermark {
		// Создаем изображение с ватермаркой во временном файле
		watermarkedTmpPath := strings.Replace(task.OriginalPath, task.ID, "watermark_"+task.ID, 1)
		err = p.addWatermark(task.OriginalPath, watermarkedTmpPath, imgSrc)
		if err != nil {
			return nil, fmt.Errorf("error add watermark: %v", err)
		}
		defer os.Remove(watermarkedTmpPath)

		img, err := os.Open(watermarkedTmpPath)
		if err != nil {
			return nil, fmt.Errorf("error open image %v", err)
		}
		imgWtr, format, err := image.Decode(img)
		if err != nil {
			return nil, fmt.Errorf("error decode image %v", err)
		}
		imgData, err = p.encodeImage(imgWtr, format, 100)
		if err != nil {
			return nil, fmt.Errorf("error encode image %v", err)
		}
		// Загружаем изображение с ватермаркой в хранилище
		watermarkedPath := fmt.Sprintf("watermarked/%s.jpg", task.ID)
		watermarkedURL, err := p.storage.Upload(ctx, watermarkedPath, imgData, "image/jpeg")
		if err != nil {
			return nil, fmt.Errorf("error upload image %v", err)
		}

		imgVariants.Watermarked = watermarkedURL
	}

	return &imgVariants, nil
}

// Закодировать изображение
func (p *ImageProcessor) encodeImage(image image.Image, format string, quality int) ([]byte, error) {
	buf := new(bytes.Buffer)

	switch format {
	case "png":
		err := png.Encode(buf, image)
		return buf.Bytes(), err
	default:
		err := jpeg.Encode(buf, image, &jpeg.Options{Quality: quality})
		return buf.Bytes(), err
	}
}

// Добавить водяной знак
func (p *ImageProcessor) addWatermark(inputFilePath string, outputFile string, image image.Image) error {
	imgWidth := image.Bounds().Dx()
	imgHeight := image.Bounds().Dy()

	quantityX := imgWidth / 100
	quantityY := imgHeight / 15

	err := gowatermark.AddWatermark(&gowatermark.Watermark{
		Image:      inputFilePath,
		OutputFile: outputFile,
		Text:       "Image Processor ",
		Color:      color.RGBA{R: 0, G: 0, B: 0, A: 80},
		Font:       gowatermark.Font{FontSize: 12},
		Align:      gowatermark.AlignCenter,
		Repeat: gowatermark.Repeat{
			RepY: quantityY,
			RepX: quantityX,
		},
		LineSpacing: 15,
		ImgSize: gowatermark.ImgSize{
			Width:  imgWidth,
			Height: image.Bounds().Dy(),
		},
	})

	return err
}
