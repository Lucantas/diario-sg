package domain

import "errors"

var (
	ErrNotFound              = errors.New("não encontrado")
	ErrInvalidEmail          = errors.New("e-mail inválido")
	ErrInvalidQuery          = errors.New("a busca deve ter entre 3 e 200 caracteres")
	ErrInvalidFilter         = errors.New("filtro inválido")
	ErrInvalidCNPJ           = errors.New("CNPJ inválido: informe 14 dígitos")
	ErrSubscriptionCancelled = errors.New("inscrição cancelada")

	ErrInvalidInput = errors.New("entrada inválida")
)
