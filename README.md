## Тестирование

# DockerContainerMonitor

## Описание

ЭтотпроектпредназначендлямониторингасостоянияDocker-контейнеров. Онвключаетвсебянесколькосервисов, которыевзаимодействуютдругсдругомдлясбораиотображенияданныхоконтейнерах.

## Структурапроекта

-**Backend**: RESTfulAPIнаGoдляработысбазойданныхPostgreSQLиаутентификации.

-**Frontend**: Веб-интерфейснаReactдляотображенияданных.

-**Pinger**: СервиснаGoдляпингаконтейнеровиотправкиданныхнаBackend.

-**PostgreSQL**: Базаданныхдляхраненияинформацииоконтейнерах.

-**nginx**: Используетсядлямаршрутизациизапросов.

-**RabbitMQ**: (Опционально) дляобработкиочередей.

-**Prometheus**: Длямониторингаметрик.

## Установкаизапуск

1.**Клонируйтерепозиторий:**

```bash

   git clone <URL вашего репозитория>

   cd <название папки>

```

2.**ЗапуститеDockerCompose:**

Убедитесь, чтоувасустановленDockerиDockerCompose.

```bash

   docker-compose up --build

```

3.**Доступкприложению:**

-Frontendдоступенпоадресу: `http://localhost`

-BackendAPIдоступенпоадресу: `http://localhost/api`

-Prometheusдоступенпоадресу: `http://localhost:9090`

## Аутентификация

-ДляполучениятокенаотправьтеPOST-запросна `/login`стелом:

```json

  {

    "username": "admin",

    "password": "password"

  }

```

-Используйтеполученныйтокендлядоступакзащищенныммаршрутам, добавляязаголовок `Authorization: Bearer <your_token>`.

## Конфигурация

-**nginx**: Конфигурациянаходитсявфайле `nginx.conf`.

-**Переменныеокружения**: Настройкибазыданныхидругихсервисовможноизменитьв `docker-compose.yml`.

### Backend

Для запуска тестов Backend-сервиса выполните:

```bash
cd backend
go test
```

### Pinger

Для запуска тестов Pinger-сервиса выполните:

```bash
cd pinger
go test
```
