# Выкат в production

Чек-лист и рекомендации по запуску Kabanos в бою. Приложение stateless
(состояние только в Postgres/Redis/объектном хранилище), поэтому API и воркер
масштабируются горизонтально, а деплой безопасен для rolling-обновлений.

Референсная отказоустойчивая топология — [`docker-compose.scale.yml`](docker-compose.scale.yml)
(Postgres primary + реплика, 3 реплики API за nginx, 2 воркера, общий Redis).

---

## 1. Секреты и конфигурация

Все значения задаются через переменные окружения (см. [`.env.example`](.env.example)).
**Никогда не коммитьте реальные секреты** — используйте секрет-менеджер (Docker/K8s
secrets, Vault, SSM).

| Переменная | Требование в проде |
|---|---|
| `APP_ENV` | `production` (включает JSON-логи, строгую валидацию) |
| `JWT_SECRET` | случайные ≥32 байта: `openssl rand -hex 32`. Общий для всех реплик API |
| `POSTGRES_DSN` | managed Postgres, `sslmode=require` |
| `POSTGRES_REPLICA_DSNS` | (опц.) реплики для тяжёлых чтений, через запятую |
| `REDIS_ADDR` / `REDIS_PASSWORD` | managed Redis с паролем/TLS |
| `PUBLIC_APP_URL` | реальный `https://…` домен фронта (идёт в ссылки писем и CORS) |
| `CORS_ALLOWED_ORIGINS` | список доверенных origin через запятую |
| `S3_*` | реальный S3/совместимое хранилище, `S3_USE_SSL=true` |
| `SMTP_*`, `MAIL_FROM_*` | реальный SMTP-провайдер (см. §3) |

> Валидация конфигурации в проде уже требует сильный `JWT_SECRET` — процесс не
> стартует с dev-значением.

---

## 2. HTTPS и заголовки

- Терминируйте TLS на балансировщике/ingress; проксируйте на API/фронт по HTTP
  внутри сети.
- Балансировщик обязан прокидывать `X-Forwarded-Proto: https` и `X-Forwarded-For`
  — по ним API отдаёт **HSTS** и вычисляет клиентский IP для rate-limit.
- Security-заголовки уже включены: API (`SecurityHeaders` middleware) и фронт
  (`next.config.mjs → headers()`): `nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy`, CSP для JSON-API, HSTS.
- Куки сессии — `httpOnly` + `Secure` (на HTTPS), токены недоступны JS (BFF).

---

## 3. Почта (важно!)

**Частая ошибка:** по умолчанию `SMTP_HOST=mailpit` — это dev-ловушка, письма
**не уходят на реальные ящики**. Воркер логирует это на старте:

```
outbound email transport smtp=mailpit:1025 tls=none …
WARN SMTP is a DEV mail catcher — email is NOT delivered to real inboxes …
```

Для прода задайте реального провайдера (Gmail/mail.ru или транзакционный сервис —
SendGrid/Mailgun/Postmark/Resend по SMTP). `SMTP_TLS` можно не указывать —
режим определится по порту (465 → TLS, 587 → STARTTLS). `MAIL_FROM_EMAIL` должен
совпадать с доменом/аккаунтом отправителя (иначе провайдер отклонит или спам).
После настройки в логе должно быть `smtp=<ваш хост>` **без** WARN.

Проверка доставки: outbox-таблица — `status='failed'` и колонка `last_error`
показывают ошибки реле; успешные — `done`.

---

## 4. База данных

- Managed Postgres 16 с автоматическими бэкапами и PITR.
- Миграции встроены в бинарь и применяются на старте под **advisory-lock**, что
  безопасно при нескольких репликах и rolling-деплое. Как альтернатива — гонять
  миграции отдельным job'ом и выключить `POSTGRES_AUTO_MIGRATE=false` на репликах.
- Реплики: заполните `POSTGRES_REPLICA_DSNS` — тяжёлые чтения уйдут на них с
  деградацией на primary при недоступности.
- Пулы соединений тюньте через `POSTGRES_MAX_CONNS/MIN_CONNS` под лимиты СУБД.

---

## 5. Объектное хранилище (фото)

- Реальный S3 или отвердённый MinIO; `S3_USE_SSL=true`.
- Бакет **приватный** — доступ только по presigned-URL (так и реализовано).
- `S3_PUBLIC_ENDPOINT` = хост, доступный браузеру (под него подписываются ссылки).

---

## 6. Масштабирование и деплой

- **API/воркер stateless** → любое число реплик за балансировщиком. Rate-limit
  общий (Redis), outbox безопасен при многих воркерах (`FOR UPDATE SKIP LOCKED`).
- **Пробы:** liveness `GET /healthz`, readiness `GET /readyz` (проверяет
  Postgres+Redis). В K8s повесьте их на соответствующие probe.
- **Graceful shutdown:** по `SIGTERM` сервер дораздаёт запросы в пределах
  `SHUTDOWN_TIMEOUT`. В K8s задайте `terminationGracePeriodSeconds` ≥ таймаута.
- Образы API/воркера — distroless (nonroot, static), без shell/пакетника.
- Фронт — Next.js standalone, тоже под несколько реплик.

---

## 7. Наблюдаемость

- В проде логи — **JSON** (`APP_ENV=production`) с `request_id`, `service`,
  статусом и латентностью. Отправляйте в Loki/Datadog/ELK.
- Точки роста (заложены архитектурно): метрики Prometheus и трейсинг OpenTelemetry
  вокруг HTTP-middleware и пула БД.
- Алерты: рост `outbox_messages.status='failed'`, 5xx-rate, латентность `/readyz`.

---

## 8. Предстартовый чек-лист

- [ ] `APP_ENV=production`, `JWT_SECRET` — сильный и общий для реплик
- [ ] Postgres: managed, `sslmode=require`, бэкапы+PITR включены
- [ ] Redis: пароль/TLS
- [ ] SMTP: реальный провайдер, в логе воркера нет WARN про dev-catcher
- [ ] `PUBLIC_APP_URL` и `CORS_ALLOWED_ORIGINS` — боевой домен
- [ ] S3: приватный бакет, TLS, корректный `S3_PUBLIC_ENDPOINT`
- [ ] TLS на LB, прокидываются `X-Forwarded-Proto`/`X-Forwarded-For`
- [ ] Настроены liveness/readiness пробы и graceful-таймаут
- [ ] Логи собираются, алерты на 5xx и failed-outbox
- [ ] Проверены регистрация → письмо → вход → основные разделы на staging
