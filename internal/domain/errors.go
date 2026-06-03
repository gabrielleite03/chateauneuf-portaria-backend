package domain

import "errors"

var (
	ErrInvalidInput = errors.New("dados invalidos")
	ErrNotFound     = errors.New("registro nao encontrado")
)
