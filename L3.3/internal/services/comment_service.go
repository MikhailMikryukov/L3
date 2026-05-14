package services

import (
	"L3.3/internal/entities"
	"context"
	"fmt"
	"strconv"
	"time"
)

type CommentsRepository interface {
	SaveComment(ctx context.Context, comment entities.Comment) error
	GetComments(ctx context.Context, id string) ([]entities.Comment, error)
	DeleteComment(ctx context.Context, id string) error
	GetAllCommentsWithLimit(ctx context.Context, limit int, offset int, sortBy string, searchQuery string) ([]entities.Comment, error)
}

// CommentService сервис работы с комментариями
type CommentService struct {
	rep CommentsRepository
}

// New создание экземпляра CommentService
func New(rep CommentsRepository) *CommentService {
	return &CommentService{rep: rep}
}

// CreateComment создание комментария
func (s *CommentService) CreateComment(author string, parent string, text string) error {
	parentID, err := strconv.Atoi(parent)
	if err != nil {
		return fmt.Errorf("некорректный parent ID: %v", err)
	}

	comment := entities.Comment{
		Author:    author,
		Parent:    parentID,
		Text:      text,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	err = s.rep.SaveComment(ctx, comment)
	if err != nil {
		return fmt.Errorf("ошибка сохранения комментария в бд: %v", err)
	}

	return nil
}

// GetCommentAndChild получает комментарий и вложенные
func (s *CommentService) GetCommentAndChild(id string) ([]entities.Comment, error) {
	ctx := context.Background()
	comments, err := s.rep.GetComments(ctx, id)
	if err != nil {
		return []entities.Comment{}, fmt.Errorf("ошибка получения комментария из бд: %v", err)
	}

	return comments, nil
}

// DeleteComment удалить комментарий из бд
func (s *CommentService) DeleteComment(id string) error {
	ctx := context.Background()
	err := s.rep.DeleteComment(ctx, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления комментария: %v", err)
	}
	return nil
}

// GetComments получить комментарии с заданным параметрами
func (s *CommentService) GetComments(limit int, offset int, sortBy string, searchQuery string) ([]entities.Comment, error) {

	ctx := context.Background()
	comments, err := s.rep.GetAllCommentsWithLimit(ctx, limit, offset, sortBy, searchQuery)
	if err != nil {
		return nil, err
	}
	return comments, nil
}
