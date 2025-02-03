FROM golang:1.21

WORKDIR /app

# Копируем только файлы go.mod и go.sum
COPY go.mod ./
COPY go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Запускаем
CMD ["./main"]