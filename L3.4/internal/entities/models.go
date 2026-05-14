package entities

const (
	// StatusPending статус ожидание обработки
	StatusPending = "pending"
	// StatusProcessing статус в обработке
	StatusProcessing = "processing"
	// StatusFailed статус неудача
	StatusFailed = "failed"
	// StatusCompleted статус успешно
	StatusCompleted = "completed"
)

// Task задание на обработку
type Task struct {
	ID           string           `json:"id"`
	Status       string           `json:"status"`
	OriginalName string           `json:"original_name"`
	OriginalPath string           `json:"original_path"`
	OriginalSize int64            `json:"original_size"`
	MIMEType     string           `json:"mime_type"`
	Params       ProcessingParams `json:"params"`
	Variants     ImageVariants    `json:"variants"`
}

// ProcessingParams параметры обработки изображения
type ProcessingParams struct {
	Quality      int   `json:"quality"`
	Width        []int `json:"width"`
	AddWatermark bool  `json:"add_watermark"`
}

// ImageVariants пути хранения обработанных изображений
type ImageVariants struct {
	Original    string         `json:"original"`
	Thumbnail   string         `json:"thumbnail"`
	Resized     map[int]string `json:"resized"`
	Watermarked string         `json:"watermarked"`
}
