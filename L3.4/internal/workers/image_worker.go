package workers

import (
	"L3.4/internal/entities"
	"L3.4/internal/services"
	"L3.4/pkg/messaging"
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

// ImageWorker воркер
type ImageWorker struct {
	kafka     *messaging.WBKafka
	db        services.TasksRepository
	processor *services.ImageProcessor
	storage   *services.StorageService
}

type processingTask struct {
	taskMsg messaging.TaskMessage
	msg     kafka.Message
}

// NewImageWorker новый воркер
func NewImageWorker(k *messaging.WBKafka, db services.TasksRepository, p *services.ImageProcessor, s *services.StorageService) *ImageWorker {
	return &ImageWorker{
		kafka:     k,
		db:        db,
		processor: p,
		storage:   s,
	}
}

// Start старт воркеров
func (w *ImageWorker) Start() error {
	log.Println("Starting kafka image worker")

	// Канал для передачи сообщений в воркер
	taskChan := make(chan processingTask, 100)

	// Запускаем пул воркеров
	for i := 0; i < 3; i++ {
		go w.worker(taskChan)
	}

	for {
		// Получаем задачу из очереди
		msg, err := w.kafka.Consumer.Fetch(context.Background())
		if err != nil {
			log.Println(err)
		}

		var taskMsg messaging.TaskMessage
		err = json.Unmarshal(msg.Value, &taskMsg)
		if err != nil {
			log.Println(err)
			// Коммитим плохие сообщения
			w.kafka.Consumer.Commit(context.Background(), msg)
			continue
		}

		// Отправляем задачу в канал для обработки
		taskChan <- processingTask{
			msg:     msg,
			taskMsg: taskMsg,
		}
	}

}

func (w *ImageWorker) worker(tasks <-chan processingTask) {
	for task := range tasks {
		w.processTask(task.taskMsg, task.msg) // ← msg передается для коммита
	}
}

func (w *ImageWorker) processTask(taskMsg messaging.TaskMessage, msg kafka.Message) {
	// В конце обработки коммитим сообщение
	defer func() {
		if err := w.kafka.Consumer.Commit(context.Background(), msg); err != nil {
			log.Printf("Failed to commit offset %d: %v", msg.Offset, err)
		}
	}()

	ctx := context.Background()

	err := w.db.UpdateStatus(ctx, taskMsg, entities.StatusProcessing, nil)
	if err != nil {
		log.Println(err)
	}

	task, err := w.db.GetTaskByID(ctx, taskMsg.ID)
	if err != nil {
		w.db.UpdateStatus(ctx, taskMsg, entities.StatusFailed, nil)
		log.Printf("не найдена задача: %v ", err)
		return
	}

	variants, err := w.processor.ProcessImage(task)
	if err != nil {
		status, errStatus := w.db.CheckStatus(ctx, taskMsg.ID)
		if errStatus != nil {
			log.Printf("ошибка проверки статуса: %v", errStatus)
		}
		if status == entities.StatusCompleted {
			return
		}

		w.db.UpdateStatus(ctx, taskMsg, entities.StatusFailed, nil)
		log.Printf("не удалось обработать: %v ", err)
		return
	}

	err = w.db.UpdateStatus(ctx, taskMsg, entities.StatusCompleted, variants)
	if err != nil {
		log.Printf("не удалось обновить статус: %v ", err)
	}

	// Удаляем временный файл
	w.storage.DeleteTemp(task.OriginalPath)

	log.Printf("задача %s выполнена", taskMsg.ID)
}
