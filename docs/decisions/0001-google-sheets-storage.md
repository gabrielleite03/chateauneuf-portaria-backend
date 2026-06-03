# ADR-0001 - Armazenamento em Google Sheets

## Status

Aprovado

## Contexto

O sistema de controle de acesso da portaria precisa armazenar registros de entrada e saída de prestadores.

O condomínio já possui uma conta institucional:

[chateauneuf.cond@gmail.com](mailto:chateauneuf.cond@gmail.com)

Também foi criada uma planilha Google Sheets para armazenamento dos dados operacionais.

## Decisão

Foi decidido utilizar:

* Google Sheets como repositório central de consulta;
* SQLite como armazenamento local;
* sincronização assíncrona entre SQLite e Google Sheets.

O frontend Angular não terá acesso direto ao Google Sheets.

Toda comunicação seguirá:

Angular → Backend Go → SQLite → Google Sheets

## Google Cloud

Foi criado um projeto Google Cloud.

APIs habilitadas:

* Google Sheets API
* Google Drive API

Foi criada uma Service Account para acesso automatizado.

A planilha foi compartilhada com a Service Account utilizando permissão Editor.

## Estrutura da planilha

Planilha:

Controle de Acesso - Portaria

Abas:

* Entradas
* Prestadores
* Moradores
* Configurações
* Logs

## Regras

* Sempre salvar primeiro no SQLite.
* Nunca depender do Google Sheets para concluir uma operação.
* Em caso de falha de internet, registrar localmente.
* Sincronizar automaticamente quando a conexão retornar.

## Motivo

Garantir:

* funcionamento offline;
* baixo custo operacional;
* facilidade de auditoria pelo conselho;
* armazenamento centralizado no Drive institucional.
