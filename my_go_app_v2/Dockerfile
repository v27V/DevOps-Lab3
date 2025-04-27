#Первая стадия сборки
# Официальный образ golang на базе Alpine используемый для сборки приложение, builder - название стадии сборки
FROM golang:alpine AS builder 
# установка зависимостей, в данном случае git
RUN apk add --no-cache git
# установка рабочей директории внутри контейнера
WORKDIR /app
# копирование файлов с зависимостями во внутрь контейнера
COPY go.mod go.sum ./
# скачивание необходимых зависимостей
RUN go mod download
# копирование всего исходного кода приложения во внутрь контейнера
COPY . .
# сбор бинарного файла без С-библиотек, с указанием сборки под linux, результат записывается в /app/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server
# второй этап сборки, финальный
FROM alpine:latest
#Добавляем корневые сертификаты для HTTPS-запросов
RUN apk add --no-cache ca-certificates
#Устанавливаем еще раз рабочую директорию внутри нового образа
WORKDIR /app
# Копируем собранный раннее бинарь из стадии "builder"
COPY --from=builder /app/server ./server
#Указываем какой порт слущать приложению
EXPOSE 8080                 
# При старте контейнера выполняем ./server
ENTRYPOINT [ "./server" ]