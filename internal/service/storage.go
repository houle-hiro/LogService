package service

import (
	"io"
	"log-service/internal/model"
)

type Storage interface {
	Save(filename string, reader io.Reader) error
	Get(filename string) (io.ReadCloser, int64, error)
	Query(prefix string) ([]model.FileInfo, error)
	Delete(filename string) error
	ListAll() ([]model.FileInfo, error)
}
