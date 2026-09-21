// Package domain contém entidades e regras de negócio puras. Não importa
// nenhuma outra camada do projeto.
package domain

import "errors"

var (
	ErrNotFound              = errors.New("não encontrado")
	ErrInvalidEmail          = errors.New("e-mail inválido")
	ErrInvalidQuery          = errors.New("a busca deve ter entre 3 e 200 caracteres")
	ErrInvalidFilter         = errors.New("filtro inválido")
	ErrSubscriptionCancelled = errors.New("inscrição cancelada")
	// ErrInvalidInput marca erros permanentes: repetir não adianta.
	ErrInvalidInput = errors.New("entrada inválida")
)
