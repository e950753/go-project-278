# Сокращатель ссылок (Go)

[![hexlet-check](https://github.com/e950753/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/e950753/go-project-278/actions)
[![CI](https://github.com/e950753/go-project-278/actions/workflows/ci.yml/badge.svg)](https://github.com/e950753/go-project-278/actions/workflows/ci.yml)

Спроектируйте приложение для удобных ссылок

Учебный проект Хекслета: https://ru.hexlet.io/programs/go


## Стек

- Go
- PostgreSQL, [sqlc](https://sqlc.dev/), [goose](https://github.com/pressly/goose)
- Caddy
- Docker
- Render
- Bugsink
- golangci-lint
- GitHub Actions

## Установка

<!-- Опишите установку: клонирование, зависимости, переменные окружения -->

```bash
git clone https://github.com/e950753/go-project-278.git
cd go-project-278
npm install
```
 
Создайте рабочую и тестовую базы данных:
 
```bash
createdb appdb
createdb appdb_test
```
 
Создайте файл `.env` в корне проекта:

 
Примените миграции:
 
```bash
set -a; source .env; set +a
go tool goose -dir db/migrations postgres "$DATABASE_URL" up
go tool goose -dir db/migrations postgres "$TEST_DATABASE_URL" up
```

## Использование

<!-- Добавьте примеры запуска и запись asciinema — именно это смотрит работодатель -->
Приложение: https://go-project-278-id1a.onrender.com
Демо: [запись в asciinema](https://asciinema.org/a/1xM4A6M3Kxq8dORN)
---

<details>
<summary>Автоматические тесты Хекслета</summary>

Тесты запускаются на каждый коммит. За запуск отвечает файл `.github/workflows/hexlet-check.yml` — не удаляйте и не переименовывайте ни его, ни репозиторий.

</details>

## О Хекслете

[Хекслет](https://ru.hexlet.io/) — школа программирования: авторские программы обучения с практикой, поддержкой наставников и реальными проектами, которые остаются в резюме. Этот репозиторий — один из таких проектов.
