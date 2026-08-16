# tg_stt_bot

Telegram-бот на Go, который расшифровывает голосовые сообщения и кружочки (video note) в текст
и отвечает reply на исходное сообщение. Расшифровка — через собственный STT-сервис parakeet
с OpenAI-совместимым API.

## Что умеет

- ловит голосовые и кружочки в чатах из whitelist и в личке админа, расшифровывает и отвечает reply;
- длинный текст режет на несколько сообщений (лимит Telegram — 4096 UTF-16 code units);
- файлы больше 20 MB не качает (ограничение Bot API `getFile`) — отвечает, что файл слишком большой;
- сообщения из чатов не из whitelist молча игнорирует;
- команды в личке админа: `/status` — проверка parakeet, `/chats` — список разрешённых чатов;
  на любую другую команду отвечает короткой подсказкой;
- записи уровня `ERROR` дополнительно уходят в сервисный чат (`service_chat_id`) отдельной горутиной,
  логи целиком — в stdout в формате JSON.

Что бот пишет в чат: расшифровку (одним или несколькими сообщениями), `Речь не распознана.` — если
сервис вернул пустой текст, `Файл слишком большой, я не могу его скачать.` — если файл больше 20 MB,
`Не удалось расшифровать сообщение.` — если скачивание, расшифровка или отправка сорвались.

Работает по long polling, четыре воркера обработки апдейтов: расшифровка занимает секунды и не должна
блокировать остальные чаты. Воркеры одновременно и предел параллелизма — хендлеры выполняются в них
(`WithNotAsyncHandlers`), иначе всплеск голосовых держал бы в памяти сколько угодно скачанных файлов.

Бота нужно добавить в чат `service_chat_id` и разрешить ему там писать: иначе ERROR-записи просто
теряются, а сбой отправки виден только в stderr.

## STT-сервис

Бот ходит в parakeet (`base_url` из конфига):

- `POST {base_url}/v1/audio/transcriptions` — multipart с полем `file` и опциональным `language`,
  ответ `{"text": "..."}`; ошибки — HTTP-код + OpenAI-конверт `{"error":{"message":"..."}}`;
- `GET {base_url}/health` — тело ответа показывается в `/status` как есть.

Конвертация форматов — на стороне сервиса (там ffmpeg), OGG/Opus и MP4 отправляются как есть.
Под капотом — модель [NVIDIA Parakeet](https://huggingface.co/nvidia/parakeet-tdt-0.6b-v2).

## Конфигурация

Конфиг читается из JSON-файла (путь — флаг `-config`, по умолчанию `config.json`), затем поверх
накатываются переменные окружения. Локальный `.env` подхватывается автоматически.
Шаблон — `config.example.json`; `config.json` с секретами в `.gitignore` и `.dockerignore`.

| Поле JSON                 | Env                       | Тип      | По умолчанию | Описание                                                              |
| ------------------------- | ------------------------- | -------- | ------------ | --------------------------------------------------------------------- |
| `app.log_level`           | `APP_LOG_LEVEL`           | string   | `INFO`       | `DEBUG`/`INFO`/`WARN`/`ERROR`, неизвестное значение → `INFO`           |
| `telegram.token`          | `TELEGRAM_TOKEN`          | string   | —            | токен бота от BotFather, обязателен                                    |
| `telegram.admin_id`       | `TELEGRAM_ADMIN_ID`       | int64    | —            | user id админа (он же chat id его лички), не может быть `0`            |
| `telegram.service_chat_id`| `TELEGRAM_SERVICE_CHAT_ID`| int64    | —            | чат/канал для ERROR-логов, не может быть `0`                           |
| `telegram.allowed_chats`  | `TELEGRAM_ALLOWED_CHATS`  | []int64  | `[]`         | whitelist чатов (в env — через запятую), нулевых элементов быть не может |
| `telegram.api_timeout`    | `TELEGRAM_API_TIMEOUT`    | duration | `30s`        | таймаут одного вызова Bot API (`getFile`, `sendMessage`), должен быть положительным |
| `telegram.download_timeout`| `TELEGRAM_DOWNLOAD_TIMEOUT`| duration| `2m`         | таймаут скачивания файла целиком, включая чтение тела контроллером      |
| `stt.base_url`            | `STT_BASE_URL`            | string   | —            | база parakeet, должна парситься как URL со схемой и хостом             |
| `stt.language`            | `STT_LANGUAGE`            | string   | `""`         | пустое значение не отправляется — сервис определяет язык сам           |
| `stt.timeout`             | `STT_TIMEOUT`             | duration | `120s`       | таймаут одного вызова STT (расшифровка, `/health`)                     |

Конфиг валидируется на старте; при ошибке валидации приложение падает с паникой на этапе `config`.
Env перекрывает файл: одна выставленная переменная (даже пустая) сильнее того, что написано в JSON.

Таймауты доезжают до внешних вызовов только через контекст: провайдер в начале каждого публичного
метода строит дочерний `context.WithTimeout` с таймаутом из конфига, у `http.Client` своего таймаута
нет — иначе дедлайнов было бы два. Все таймауты — положительные duration (`0` не принимается).

## Локальный запуск

Нужен Go той же версии, что в `go.mod` (сейчас 1.26.5) или новее.

```sh
cp config.example.json config.json
# заполнить token, admin_id, service_chat_id, allowed_chats, base_url
make run                      # go run ./cmd/app с config.json
make build && ./bin/tg_stt_bot -config config.json
```

Из LAN до кластера удобно перекрывать адрес сервиса переменной окружения:

```sh
STT_BASE_URL=http://192.168.10.53:5092 make run
```

Чтобы бот видел голосовые в группах, у бота должен быть выключен privacy mode у BotFather
(или выдана админка в чате).

## Разработка

```sh
make lint      # golangci-lint (ставится в ./bin)
make test      # go test -race ./... с кешами в ./.cache
make format    # go fmt ./...
make tidy      # go mod tidy
```

Правила кода и архитектуры — в [AGENTS.md](AGENTS.md).

## Docker

```sh
docker build -t tg_stt_bot .
docker run --rm -v "$PWD/config.json:/etc/tg-stt-bot/config.json:ro" \
  tg_stt_bot -config /etc/tg-stt-bot/config.json
```

Образ — multi-stage: сборка на `golang:1.26.6`, рантайм `gcr.io/distroless/static-debian12:nonroot`,
внутри только бинарник; конфиг монтируется снаружи (в k8s — Secret с целым `config.json`).
Архитектура берётся из `TARGETOS`/`TARGETARCH`, так что `docker buildx --platform` работает как ожидается.
В образ копируются только `cmd/` и `internal/`: новая директория с кодом верхнего уровня сломает сборку
образа, оставив `make lint`/`make test` зелёными — её нужно добавить в `Dockerfile` руками.

## CI/CD

- `.github/workflows/verify.yml` — переиспользуемый workflow с джобами `lint`, `test`, `build`;
- `.github/workflows/ci.yml` — на push и pull request запускает verify;
- `.github/workflows/release.yml` — ручной запуск (`workflow_dispatch`) только с `master`: verify,
  затем сборка и пуш образа в `ghcr.io/1kovalevskiy/tg_stt_bot` с тегами `YYYYMMDD-HHMM-<short-sha>`
  и `latest`.

## Структура

```
cmd/app/            точка входа и вайринг (единственное место, где известны конкретные типы)
internal/configs/   загрузка и валидация конфига, getter-API
internal/models/    чистые модели, функции (разбивка текста, проверка допуска чата) и все константы
internal/providers/ stt (parakeet HTTP) и telegram (Bot API)
internal/controllers/
  chat-controller/  сценарий расшифровки голосовых и кружочков
  admin-controller/ команды админа
```
