package domain

import "errors"

var (
	ErrInvalidInput               = errors.New("dados invalidos")
	ErrNotFound                   = errors.New("registro nao encontrado")
	ErrReservationDateUnavailable = errors.New("data ja possui uma reserva ativa")
	ErrCancellationDeadline       = errors.New("cancelamento permitido somente ate 7 dias antes do evento")
	ErrActiveReservationDeletion  = errors.New("uma reserva ativa deve ser cancelada antes de ser excluida")
	ErrActiveVisitExists          = errors.New("visitante ja possui uma entrada ativa; registre a saida antes de uma nova entrada")
)
