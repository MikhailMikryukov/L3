package handlers

import (
	"L3.4/internal/entities"
	"L3.4/internal/services"
	"L3.4/pkg/messaging"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrorResponse ответ с ошибкой
type errorResponse struct {
	Error string `json:"error"`
}

// ImageResponse ответ со ссылками на изображения
type ImageResponse struct {
	Images []imageURL `json:"images,omitempty"`
	Errors []string   `json:"errors,omitempty"`
}
type imageURL struct {
	Variant string `json:"variant"`
	URL     string `json:"URL"`
}

// ImageProcessorHandler обработчик запросов
type ImageProcessorHandler struct {
	processor *services.ImageProcessor
	db        services.TasksRepository
	storage   *services.StorageService
	kafka     *messaging.WBKafka
}

// New создает экземпляр ImageProcessorHandler
func New(p *services.ImageProcessor, db services.TasksRepository, s *services.StorageService, k *messaging.WBKafka) *ImageProcessorHandler {
	return &ImageProcessorHandler{
		processor: p,
		db:        db,
		storage:   s,
		kafka:     k,
	}
}

// NewRouter создание и настройка обработчика
func NewRouter(p *services.ImageProcessor, db services.TasksRepository, s *services.StorageService, k *messaging.WBKafka) *http.ServeMux {
	mux := http.NewServeMux()
	imageHandler := New(p, db, s, k)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})
	// Загрузка изображения на обработку
	mux.HandleFunc("/upload", imageHandler.upload)
	// Получение и удаление изображений
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		// Вызываем обработчик
		if r.Method == "GET" {
			imageHandler.getImage(w, r)
		}
		if r.Method == "DELETE" {
			imageHandler.delete(w, r)
		}
	})
	// Получение статуса
	mux.HandleFunc("/status", imageHandler.getStatus)

	return mux
}

// upload загрузка изображения
func (h *ImageProcessorHandler) upload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("image")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	// Проверяем размер файла
	if header.Size > 10*1024*1024 {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("слишком большой файл (максимум 10 MB)").Error())
		return
	}

	// Проверяем MIME тип
	mimeType := header.Header.Get("Content-type")
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("неподходящий тип. загрузите jpeg, png, gif").Error())
		return
	}

	// Открываем файл
	fileContent, err := header.Open()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("не удалось открыть файл").Error())
		return
	}
	defer fileContent.Close()

	// Читаем содержимое
	buffer := make([]byte, header.Size)
	_, err = fileContent.Read(buffer)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("не удалось прочитать файл").Error())
		return
	}

	// Заполняем параметры для обработки изображения
	var processingParams entities.ProcessingParams
	// Значение качества
	processingParams.Quality = 70

	// Получаем ширину для ресайза
	widthsStr := r.FormValue("widths")
	if widthsStr != "" {
		widths := strings.Split(widthsStr, ",")
		for _, v := range widths {
			width, err := strconv.Atoi(v)
			if err != nil {
				h.writeError(w, http.StatusInternalServerError, fmt.Errorf("некорректно указана ширина").Error())
				return
			}
			processingParams.Width = append(processingParams.Width, width)
		}
	}

	// Нужна ли вотермарка
	watermark := r.FormValue("watermark")
	if watermark == "true" {
		processingParams.AddWatermark = true
	}

	// Генерируем имя для временного файла
	ext := ".jpg"
	if strings.Contains(mimeType, "png") {
		ext = ".png"
	} else if strings.Contains(mimeType, "gif") {
		ext = ".gif"
	}

	id := uuid.New().String()
	tempFileName := id + ext

	// Сохраняем временную копию
	tempFilePath, err := h.storage.SaveTemp(buffer, tempFileName)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("не удалось сохранить временный файл").Error())
		return
	}

	// Задача для сохранения в бд
	task := &entities.Task{
		ID:           id,
		Status:       entities.StatusPending,
		OriginalName: header.Filename,
		OriginalPath: tempFilePath,
		OriginalSize: header.Size,
		MIMEType:     mimeType,
		Params:       processingParams,
		Variants:     entities.ImageVariants{},
	}
	// Сохраняем в бд
	ctx := context.Background()
	if err := h.db.Create(ctx, task); err != nil {
		log.Println(err)
		h.storage.DeleteTemp(tempFilePath) // чистим временный файл
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("не удалось собрать таску").Error())
		return
	}
	// Готовим сообщение для kafka
	paramsJSON, _ := json.Marshal(processingParams)
	taskMessage := messaging.TaskMessage{
		ID:           task.ID,
		OriginalPath: task.OriginalPath,
		Params:       paramsJSON,
	}

	messageValue, err := json.Marshal(taskMessage)
	if err != nil {
		h.storage.DeleteTemp(tempFilePath)
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("не удалось собрать сообщение в кафку").Error())
	}

	// Отправляем сообщение в kafka
	err = h.kafka.Producer.Send(context.Background(), []byte(taskMessage.ID), messageValue)
	if err != nil {
		log.Printf("не удалось отправить сообщение в kafka: %v", err)
	}

	h.writeJSON(w, http.StatusOK, task)
}

// getImage получить изображение
func (h *ImageProcessorHandler) getImage(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	watermark := r.FormValue("watermark")
	origin := r.FormValue("original")
	widths := r.FormValue("widths")
	thumb := r.FormValue("thumbnail")

	ctx := context.Background()
	task, err := h.db.GetTaskByID(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, fmt.Errorf("изображение не найдено").Error())
		return
	}

	if task.Status != entities.StatusCompleted {
		h.writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  task.Status,
			"message": "изображение еще обрабатывается",
		})
		return
	}

	var result []imageURL
	var errors []string

	if watermark == "true" {
		if task.Variants.Watermarked == "" {
			errors = append(errors, fmt.Sprintf("нет этого изображения с ватермаркой, ID: %v", task.ID))
		}
		URL, err := h.storage.GetDownloadURL(ctx, task.Variants.Watermarked, 10*time.Minute)
		if err != nil {
			log.Println(err)
		}

		result = append(result, imageURL{Variant: "watermark", URL: URL})
	}

	if origin == "true" {

		URL, err := h.storage.GetDownloadURL(ctx, task.Variants.Original, 10*time.Minute)
		if err != nil {
			log.Println(err)
		}
		result = append(result, imageURL{Variant: "original", URL: URL})
	}

	if thumb == "true" {
		if task.Variants.Thumbnail == "" {
			errors = append(errors, fmt.Sprintf("нет миниатюры для этого изображения, ID: %v", task.ID))
		}

		URL, err := h.storage.GetDownloadURL(ctx, task.Variants.Thumbnail, 10*time.Minute)
		if err != nil {
			log.Println(err)
		}
		result = append(result, imageURL{Variant: "thumbnail", URL: URL})
	}

	if widths != "" {
		split := strings.Split(widths, ",")
		for _, s := range split {
			resized, err := strconv.Atoi(s)
			if err != nil {
				errors = append(errors, fmt.Sprintf("некорректное значение размера: %v", s))
				continue
			}

			if _, ok := task.Variants.Resized[resized]; !ok {
				errors = append(errors, fmt.Sprintf("нет изображения с размером %v, ID: %v", resized, task.ID))
				continue
			}

			URL, err := h.storage.GetDownloadURL(ctx, task.Variants.Resized[resized], 10*time.Minute)
			if err != nil {
				log.Println(err)
			}
			result = append(result, imageURL{Variant: s, URL: URL})
		}
	}

	if len(errors) > 0 {
		h.writeJSON(w, http.StatusInternalServerError, ImageResponse{
			Images: result,
			Errors: errors,
		})
		return
	}

	h.writeJSON(w, http.StatusOK, ImageResponse{
		Images: result,
	})
}

// Delete получить изображение
func (h *ImageProcessorHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	ctx := context.Background()
	task, err := h.db.GetTaskByID(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, fmt.Errorf("изображение не найдено").Error())
		return
	}

	// Собираем файлы для удаления
	var filesToDelete []string

	if task.Variants.Thumbnail != "" {
		filesToDelete = append(filesToDelete, task.Variants.Thumbnail)
	}

	if task.Variants.Original != "" {
		filesToDelete = append(filesToDelete, task.Variants.Original)
	}

	if task.Variants.Watermarked != "" {
		filesToDelete = append(filesToDelete, task.Variants.Watermarked)
	}

	if len(task.Variants.Resized) > 0 {
		for _, resized := range task.Variants.Resized {
			filesToDelete = append(filesToDelete, resized)
		}
	}

	// Удаляем
	var errors []errorResponse
	for _, f := range filesToDelete {
		err := h.storage.Delete(context.Background(), f)
		if err != nil {
			errors = append(errors, errorResponse{Error: fmt.Errorf("не удален файл").Error()})
		}
	}

	// Удаляем из бд
	err = h.db.Delete(ctx, task)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("не удалось удалить изображение из бд").Error())
		log.Println(err)
		return
	}

	if len(errors) > 0 {
		w.WriteHeader(http.StatusPartialContent)
		json.NewEncoder(w).Encode(errors)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"response": "изображение удалено",
	})
}

// getInformation получить изображение
func (h *ImageProcessorHandler) getStatus(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	ctx := context.Background()
	status, err := h.db.CheckStatus(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, fmt.Errorf("не удалось получить статус").Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": status,
	})
}

func (h *ImageProcessorHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *ImageProcessorHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorResponse{Error: message})
}
