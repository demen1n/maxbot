# Спека MAX Bot API

`max-bot-api-0.0.33.json` — OpenAPI 3.0 спека MAX Bot API, снята 2026-09-04. Это источник
истины для этой библиотеки.

## Откуда она

MAX не публикует спеку отдельным файлом. На `https://dev.max.ru/docs-api` документация —
Next.js-приложение, которое рендерит схемы из JSON, вшитого в один из JS-чанков бандла
(на момент снятия — `/_next/static/chunks/5388-42761ac79dbbff34.js`, литерал `JSON.parse('…')`).
`extract_spec.py` находит нужный чанк по странице `/docs-api`, распаковывает JS-литерал и
сохраняет результат:

```bash
python3 spec/extract_spec.py            # -> spec/max-bot-api-<version>.json
```

Имя файла содержит `info.version`, так что новая версия ляжет рядом, а не поверх — diff двух
файлов и показывает, что MAX изменил.

## Почему не `schema.yaml` из официальной либы

`max-messenger/max-bot-api-client-go/schemes/schema.yaml` — версия **0.0.10** с устаревшим
хостом `botapi.max.ru`: в ней нет Comments API, нет `PATCH /me/commands`, старый `CallbackAnswer`
с полем `notification`. Зато в ней есть `ReplyKeyboardAttachment`, `ChatButton` и
`message_chat_created`, которых нет в сайдбаре сайта — эти объекты в 0.0.33 остались в
`components.schemas`, но выпали из discriminator-маппингов. Держать оба источника в голове
приходится, но приоритет — у 0.0.33.

Код самой официальной либы местами отстаёт и от 0.0.10 (у `ChatMember` нет `first_name`/`alias`,
у `ChatPatch` — `description`, enum прав урезан), поэтому эталоном поведения он не является.
